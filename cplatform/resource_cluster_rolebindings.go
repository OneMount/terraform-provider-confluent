package cplatform

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	confluent "github.com/OneMount/gonfluent"
)

var (
	validRole = []string{
		"AuditAdmin",
		"ClusterAdmin",
		"DeveloperManage",
		"DeveloperRead",
		"DeveloperWrite",
		"Operator",
		"ResourceOwner",
		"SecurityAdmin",
		"SystemAdmin",
		"UserAdmin",
	}
	scopeRole = []string{
		"DeveloperRead",
		"DeveloperWrite",
		"DeveloperManage",
		"ResourceOwner",
	}
	validCluster = []string{
		"Kafka",
		"SchemaRegistry",
		"Connect",
		"KSQL",
	}
	validPatternType = []string{
		"LITERAL",
		"PREFIXED",
	}
)

// roleBindings will bind the roles to users/scope
// example:
// resource "cluster_role_binding" "user_confluent_test" {
//   principal  = "User:confluent-test"
//   role 		= "Operation"
//	 cluster_id = "zxvdlaskdjal"
// 	 cluster_type = "Kafka"
//
//   provider = confluent-kafka.confluent
//	}
// Resource ID = cluster_type + "|" + clusterId + "|" + principal + "|" + role
func clusterRoleBindings() *schema.Resource {
	return &schema.Resource{
		CreateContext: clusterRoleBindingsCreate,
		DeleteContext: clusterRoleBindingsDelete,
		ReadContext:   clusterRoleBindingsRead,

		Schema: map[string]*schema.Schema{
			"principal": {
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
				Description: "Enable communication with the Kafka Cluster over TLS.",
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					v := val.(string)
					if !strings.Contains(v, "User:") || !strings.Contains(v, "Group:") && strings.Contains(v, "|") {
						errs = append(errs, fmt.Errorf("%q must be defined with USER: or GROUP: and must not have |, got: %s", key, v))
					}
					return
				},
			},
			"role": {
				Type:         schema.TypeString,
				ForceNew:     true,
				Required:     true,
				Description:  "Role of",
				DefaultFunc:  schema.EnvDefaultFunc("ROLE", "DeveloperRead"),
				ValidateFunc: validation.StringInSlice(validRole, false),
			},
			"cluster_type": {
				Type:         schema.TypeString,
				ForceNew:     true,
				Required:     true,
				ValidateFunc: validation.StringInSlice(validCluster, false),
			},
			"cluster_id": {
				Type:        schema.TypeString,
				ForceNew:    true,
				Required:    true,
				Description: "The ID of cluster",
			},
			"schema_registry_cluster_id": {
				Type:        schema.TypeString,
				ForceNew:    true,
				Optional:    true,
				Description: "The ID of cluster",
			},
			"connect_cluster_id": {
				Type:        schema.TypeString,
				ForceNew:    true,
				Optional:    true,
				Description: "The ID of cluster",
			},
			"ksql_cluster_id": {
				Type:        schema.TypeString,
				ForceNew:    true,
				Optional:    true,
				Description: "The ID of cluster",
			},
		},
	}
}

func clusterRoleBindingsRead(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*confluent.Client)

	var (
		clusterType string
		subClusterId string
	)

	f, err := filterClusterTypeWithClusterId(d)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterId := parseIdToResourcesList(d.Id())[1]
	principal := parseIdToResourcesList(d.Id())[2]
	role := parseIdToResourcesList(d.Id())[3]

	t := strings.Split(parseIdToResourcesList(d.Id())[0], ":")
	if len(t) < 2 {
		clusterType = parseIdToResourcesList(d.Id())[0]
	} else {
		clusterType = t[0]
		subClusterId = t[1]
	}

	cd := confluent.ClusterDetails{}
	cd.Clusters.KafkaCluster = clusterId

	switch clusterType {
	case "Kafka":

	case "SchemaRegistry":
		if !(contains(f, "schema_registry_cluster_id")) {
			return diag.FromErr(fmt.Errorf("miss parameter: schema_registry_cluster_id"))
		}
		cd.Clusters.SchemaRegistryCluster = subClusterId
	case "Connect":
		if !(contains(f, "connect_cluster_id")) {
			return diag.FromErr(fmt.Errorf("miss parameter: connect_cluster_id"))
		}
		cd.Clusters.ConnectCluster = subClusterId
	case "KSQL":
		if !(contains(f, "ksql_cluster_id")) {
			return diag.FromErr(fmt.Errorf("miss parameter: ksql_cluster_id"))
		}
		cd.Clusters.KSqlCluster = subClusterId
	}

	_, err = c.LookupRoleBinding(principal, role, cd)
	if err != nil {
		log.Printf("[ERROR] Error getting role-binding %s from Confluent", err)
		return diag.FromErr(err)
	}

	return nil
}

func clusterRoleBindingsCreate(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*confluent.Client)

	f, err := filterClusterTypeWithClusterId(d)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterId := d.Get("cluster_id").(string)
	principal := d.Get("principal").(string)
	role := d.Get("role").(string)
	clusterType := d.Get("cluster_type").(string)
	cd := confluent.ClusterDetails{}
	cd.Clusters.KafkaCluster = clusterId

	var rId string
	switch clusterType {
	case "Kafka":
		rId = clusterType + "|" + clusterId + "|" + principal + "|" + role
	case "SchemaRegistry":
		if !(contains(f, "schema_registry_cluster_id")) {
			return diag.FromErr(fmt.Errorf("miss parameter: schema_registry_cluster_id"))
		}
		cd.Clusters.SchemaRegistryCluster = d.Get("schema_registry_cluster_id").(string)
		rId = clusterType + ":" + d.Get("schema_registry_cluster_id").(string) + "|" + clusterId + "|" + principal + "|" + role
	case "Connect":
		if !(contains(f, "connect_cluster_id")) {
			return diag.FromErr(fmt.Errorf("miss parameter: connect_cluster_id"))
		}
		cd.Clusters.ConnectCluster = d.Get("connect_cluster_id").(string)
		rId = clusterType + ":" + d.Get("connect_cluster_id").(string) + "|" + clusterId + "|" + principal + "|" + role
	case "KSQL":
		if !(contains(f, "ksql_cluster_id")) {
			return diag.FromErr(fmt.Errorf("miss parameter: ksql_cluster_id"))
		}
		cd.Clusters.KSqlCluster = d.Get("ksql_cluster_id").(string)
		rId = clusterType + ":" + d.Get("ksql_cluster_id").(string) + "|" + clusterId + "|" + principal + "|" + role
	}

	err = c.BindPrincipalToRole(principal, role, cd)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(rId)
	return nil
}

func clusterRoleBindingsDelete(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*confluent.Client)
	var (
		clusterType string
		subClusterId string
	)

	f, err := filterClusterTypeWithClusterId(d)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterId := parseIdToResourcesList(d.Id())[1]
	principal := parseIdToResourcesList(d.Id())[2]
	role := parseIdToResourcesList(d.Id())[3]

	t := strings.Split(parseIdToResourcesList(d.Id())[0], ":")
	if len(t) < 2 {
		clusterType = parseIdToResourcesList(d.Id())[0]
	} else {
		clusterType = t[0]
		subClusterId = t[1]
	}

	cd := confluent.ClusterDetails{}
	cd.Clusters.KafkaCluster = clusterId

	switch clusterType {
	case "Kafka":
	case "SchemaRegistry":
		if !(contains(f, "schema_registry_cluster_id")) {
			return diag.FromErr(fmt.Errorf("miss parameter: schema_registry_cluster_id"))
		}
		cd.Clusters.SchemaRegistryCluster = subClusterId
	case "Connect":
		if !(contains(f, "connect_cluster_id")) {
			return diag.FromErr(fmt.Errorf("miss parameter: connect_cluster_id"))
		}
		cd.Clusters.ConnectCluster = subClusterId
	case "KSQL":
		if !(contains(f, "ksql_cluster_id")) {
			return diag.FromErr(fmt.Errorf("miss parameter: ksql_cluster_id"))
		}
		cd.Clusters.KSqlCluster = subClusterId
	}

	err = c.DeleteRoleBinding(principal, role, cd)
	if err != nil {
		diag.FromErr(err)
	}

	return nil
}

func parseIdToResourcesList(bindId string) []string {
	return strings.Split(bindId, "|")
}

func filterClusterTypeWithClusterId(d *schema.ResourceData) ([]string, error) {
	var k []string
	if d.Get("schema_registry_cluster_id").(string) != "" {
		k = append(k, "schema_registry_cluster_id")
	}
	if d.Get("connect_cluster_id").(string) != "" {
		k = append(k, "connect_cluster_id")
	}
	if d.Get("ksql_cluster_id").(string) != "" {
		k = append(k, "ksql_cluster_id")
	}

	if len(k) > 1 {
		return nil, fmt.Errorf("cannot specific schema/connect/ksql at the same time")
	}

	return k, nil
}

func contains(slice []string, item string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}

	_, ok := set[item]
	return ok
}
