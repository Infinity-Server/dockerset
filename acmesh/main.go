package main

import (
  "os"
  "fmt"
  "context"
  "encoding/json"

  "k8s.io/klog"
  "k8s.io/client-go/rest"
  "k8s.io/client-go/kubernetes"
  extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"

  "github.com/jetstack/cert-manager/pkg/acme/webhook/cmd"
  "github.com/jetstack/cert-manager/pkg/issuer/acme/dns/util"
  "github.com/jetstack/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
  defaultTTL = 600
  acmeDelegate = "/root/acme_delegate"
)

var GroupName = os.Getenv("GROUP_NAME")

func main() {
  if GroupName == "" {
    panic("GROUP_NAME must be specified")
  }

  cmd.RunWebhookServer(GroupName,
  &customDNSProviderSolver{},
)
}

type customDNSProviderSolver struct {
  client *kubernetes.Clientset
}

type envSecretRef struct {
  Name          string                  `json:"name"`
  Namespace     string                  `json:"namespace"`
}

type customDNSProviderConfig struct {
  TTL           *uint64                 `json:"ttl"`
  DNSAPI        string                  `json:"dnsapi"`
  EnvSecretRef  envSecretRef            `json:"env"`
}

type envFromSecret []string

func (c *customDNSProviderSolver) Name() string {
  return "acmesh"
}

func (c *customDNSProviderSolver) DoDNSAPI(action string, ch *v1alpha1.ChallengeRequest) error {
  cfg, err := loadConfig(ch.Config)
  if err != nil {
    klog.Errorf("Failed to log config %v: %v", ch.Config, err)
    return err
  }

  envSecret, err := c.client.CoreV1().Secrets(cfg.EnvSecretRef.Namespace).Get(context.TODO(), cfg.EnvSecretRef.Name, metav1.GetOptions{})
  if err != nil {
    return err
  }

  envData, ok := envSecret.Data["env"]
  if !ok {
    return fmt.Errorf("no env in secret")
  }

  env := envFromSecret{}
  if err := json.Unmarshal(envData, &env); err != nil {
    return err
  }

  procAttr := &os.ProcAttr{
    Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
    Env: env,
  }
  process, err := os.StartProcess(acmeDelegate, []string{
    acmeDelegate, cfg.DNSAPI, "add", util.UnFqdn(ch.ResolvedFQDN), ch.Key,
  }, procAttr)
  if err != nil {
    return err
  }

  process.Wait()
  return nil
}

func (c *customDNSProviderSolver) Present(ch *v1alpha1.ChallengeRequest) error {
  return c.DoDNSAPI("add", ch)
}

func (c *customDNSProviderSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
  return c.DoDNSAPI("rm", ch)
}

func (c *customDNSProviderSolver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
  cl, err := kubernetes.NewForConfig(kubeClientConfig)
  if err != nil {
    klog.Errorf("Failed to new kubernetes client: %v", err)
    return err
  }
  c.client = cl
  return nil
}

func loadConfig(cfgJSON *extapi.JSON) (customDNSProviderConfig, error) {
  ttl := uint64(defaultTTL)
  cfg := customDNSProviderConfig{TTL: &ttl}
  if cfgJSON == nil {
    return cfg, nil
  }
  if err := json.Unmarshal(cfgJSON.Raw, &cfg); err != nil {
    return cfg, fmt.Errorf("error decoding solver config: %v", err)
  }

  return cfg, nil
}
