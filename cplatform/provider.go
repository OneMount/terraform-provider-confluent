package cplatform

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	confluent "github.com/OneMount/gonfluent"
)

var (
	diags diag.Diagnostics
)

const (
	ClientVersion = "0.1"
	UserAgent     = "confluent-client-go-sdk-" + ClientVersion
)

func Provider() *schema.Provider {
	log.Printf("[INFO] Creating Provider")
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"username": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("CONFLUENT_USERNAME", ""),
			},
			"password": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("CONFLUENT_PASSWORD", ""),
			},
			"bootstrap_servers": {
				Type:        schema.TypeList,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Required:    true,
				Description: "A list of kafka brokers",
			},
			"ca_cert_file": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KAFKA_CA_CERT", nil),
				Description: "Path to a CA certificate file to validate the server's certificate.",
				Deprecated:  "This parameter is now deprecated and will be removed in a later release, please use `ca_cert` instead.",
			},
			"client_cert_file": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KAFKA_CLIENT_CERT", nil),
				Description: "Path to a file containing the client certificate.",
				Deprecated:  "This parameter is now deprecated and will be removed in a later release, please use `client_cert` instead.",
			},
			"client_key_file": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KAFKA_CLIENT_KEY", nil),
				Description: "Path to a file containing the private key that the certificate was issued for.",
				Deprecated:  "This parameter is now deprecated and will be removed in a later release, please use `client_key` instead.",
			},
			"ca_cert": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KAFKA_CA_CERT", nil),
				Description: "CA certificate file to validate the server's certificate.",
			},
			"client_cert": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KAFKA_CLIENT_CERT", nil),
				Description: "The client certificate.",
			},
			"client_key": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KAFKA_CLIENT_KEY", nil),
				Description: "The private key that the certificate was issued for.",
			},
			"client_key_passphrase": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KAFKA_CLIENT_KEY_PASSPHRASE", nil),
				Description: "The passphrase for the private key that the certificate was issued for.",
			},
			"sasl_username": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KAFKA_SASL_USERNAME", nil),
				Description: "Username for SASL authentication.",
			},
			"sasl_password": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KAFKA_SASL_PASSWORD", nil),
				Description: "Password for SASL authentication.",
			},
			"sasl_mechanism": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KAFKA_SASL_MECHANISM", "plain"),
				Description: "SASL mechanism, can be plain, scram-sha512, scram-sha256",
			},
			"skip_tls_verify": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KAFKA_SKIP_VERIFY", "false"),
				Description: "Set this to true only if the target Kafka server is an insecure development instance.",
			},
			"tls_enabled": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("KAFKA_ENABLE_TLS", "true"),
				Description: "Enable communication with the Kafka Cluster over TLS.",
			},
			"timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     120,
				Description: "Timeout in seconds",
			},
		},
		ConfigureContextFunc: providerConfigure,
		ResourcesMap: map[string]*schema.Resource{
			"kafka_topic":          topics(),
			"cluster_role_binding": clusterRoleBindings(),
			"kafka_topic_rbac":     kafkaTopicRBAC(),
			"schema_registry_rbac": schemaRegistryRBAC(),
			"connectors_rbac":      connectorsRBAC(),
		},
	}
}
func providerConfigure(_ context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	log.Printf("[INFO] Initializing Confluent Platform client")
	username := d.Get("username").(string)
	password := d.Get("password").(string)
	brokers := dTos("bootstrap_servers", d)

	var diags diag.Diagnostics
	kConfig := &confluent.Config{
		BootstrapServers: brokers,
		CACert:           d.Get("ca_cert").(string),
		ClientCert:       d.Get("client_cert").(string),
		ClientCertKey:    d.Get("client_key").(string),
		SkipTLSVerify:    d.Get("skip_tls_verify").(bool),
		SASLMechanism:    d.Get("sasl_mechanism").(string),
		TLSEnabled:       d.Get("tls_enabled").(bool),
		Timeout:          d.Get("timeout").(int),
	}
	kClient, sarama, err := confluent.NewDefaultSaramaClient(kConfig)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	saramaAdmin, err := confluent.NewDefaultSaramaClusterAdmin(sarama)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	for _, v := range d.Get("bootstrap_servers").([]interface{}) {
		baseUrl := "https://" + strings.Replace(v.(string), "9093", "8090", 1)
		httpClient := confluent.NewDefaultHttpClient(baseUrl, username, password)
		client := confluent.NewClient(httpClient, kClient, saramaAdmin)
		httpClient.UserAgent = UserAgent
		bearerToken, err := client.Login()
		if err == nil {
			httpClient.Token = bearerToken
			return client, diags
		}

		log.Printf("[WARN] Cannot login to confluent server " + baseUrl + ". Retry with another server!.")

	}

	return nil, diag.FromErr(fmt.Errorf("cannot find any Kafka Nodes available in the list of bootstrap server"))
}

func dTos(key string, d *schema.ResourceData) *[]string {
	var r *[]string

	if v, ok := d.GetOk(key); ok {
		if v == nil {
			return r
		}
		vI := v.([]interface{})
		b := make([]string, len(vI))

		for i, vv := range vI {
			if vv == nil {
				log.Printf("[DEBUG] %d %v was nil", i, vv)
				continue
			}
			b[i] = vv.(string)
		}
		r = &b
	}

	return r
}