package main

import (
  "os"
  "fmt"
  "context"
  "strings"
  "encoding/json"
  "k8s.io/client-go/kubernetes"

  "k8s.io/klog"
  "k8s.io/client-go/rest"
  metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
  extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"

  "github.com/jetstack/cert-manager/pkg/acme/webhook/cmd"
  "github.com/jetstack/cert-manager/pkg/issuer/acme/dns/util"
  "github.com/jetstack/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"

  "github.com/google/uuid"
)

const (
  defaultTTL = 600
  acmeDelegate = "/root/acme_delegate"
  acmeReturnValue = "ACME_RETVAL"
)

var GroupName = os.Getenv("GROUP_NAME")

func main() {
  if GroupName == "" {
    panic("GROUP_NAME must be specified")
  }

  cmd.RunWebhookServer(GroupName, &customDNSProviderSolver{})
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

  uuid := uuid.New()
  stdoutFile, err := os.CreateTemp("/tmp", uuid.String())
  defer os.Remove(stdoutFile.Name())

  procAttr := &os.ProcAttr{
    Files: []*os.File{os.Stdin, stdoutFile, os.Stderr},
    Env: env,
  }

  process, err := os.StartProcess(acmeDelegate, []string{
    acmeDelegate, cfg.DNSAPI, action, util.UnFqdn(ch.ResolvedFQDN), ch.Key,
  }, procAttr)
  if err != nil {
    return err
  }

  process.Wait()
  stdoutFile.Sync()

  outFile, err := os.Open(stdoutFile.Name())
  if err != nil {
    return err
  }

  output := make([]byte, 1048576)
  count, err := outFile.Read(output)
  if err != nil {
    return err
  }

  os.Stdout.WriteString(string(output) + "\n")
  os.Stdout.WriteString("[ACME] read output count=" + fmt.Sprint(count) + "\n")
  lines := strings.Split(string(output), "\n")

  retval := "0"
  for _, line := range lines {
    if strings.HasPrefix(line, acmeReturnValue) {
      items := strings.Split(line, acmeReturnValue)
      retval = items[1]
    }
  }

  if retval == "0" {
    return nil
  }
  
  return fmt.Errorf("Failed to run acme.sh, error=%s ...", retval)
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
