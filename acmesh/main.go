package main

import (
	"encoding/json"
	"fmt"
	"github.com/jetstack/cert-manager/pkg/issuer/acme/dns/util"
	"k8s.io/client-go/kubernetes"
	"os"
	"strings"

	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/client-go/rest"
	"k8s.io/klog"

	"github.com/jetstack/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/jetstack/cert-manager/pkg/acme/webhook/cmd"
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

type customDNSProviderConfig struct {
	TTL          *uint64                  `json:"ttl"`
	DNSAPI       string                   `json:"dnsapi"`
  Env          []string                 `json:"env"`
}

func (c *customDNSProviderSolver) Name() string {
	return "acmesh"
}


func extractRecordName(fqdn, zone string) string {
	if idx := strings.Index(fqdn, "."+zone); idx != -1 {
		return fqdn[:idx]
	}

	return util.UnFqdn(fqdn)
}

func (c *customDNSProviderSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		klog.Errorf("Failed to log config %v: %v", ch.Config, err)
		return err
	}

  procAttr := &os.ProcAttr{
    Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
    Env: cfg.Env,
  }
  process, err := os.StartProcess(acmeDelegate, []string{
    acmeDelegate, cfg.DNSAPI, "add", extractRecordName(ch.ResolvedFQDN, ch.ResolvedZone), ch.Key,
  }, procAttr)
  if err != nil {
    return err
  }

  process.Wait()
	return nil
}

func (c *customDNSProviderSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		klog.Errorf("Failed to log config %v: %v", ch.Config, err)
		return err
	}

  procAttr := &os.ProcAttr{
    Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
    Env: cfg.Env,
  }
  process, err := os.StartProcess(acmeDelegate, []string{
    acmeDelegate, cfg.DNSAPI, "rm", extractRecordName(ch.ResolvedFQDN, ch.ResolvedZone), ch.Key,
  }, procAttr)
  if err != nil {
    return err
  }

  process.Wait()
	return nil
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
