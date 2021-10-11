# Terraform Confluent Platform Provider

This is provider for Confluent Platform, not Confluent Cloud. If you're seeking the Confluent Cloud provider, please visit [This project](https://github.com/Mongey/terraform-provider-confluentcloud). 

But if you're seeking the terraform provider to implement IaC for your hosted Confluent Platform, this project is exactly what you need.

## Getting started

### 1. Installation

- Clone this project

```
git clone git@github.com:wayarmy/gonfluent.git && cd gonfluent
```

- Build the binary:

```
make prepare
make compile
```

- Find your own binary in the `bin` folder. For example, if you're running `terraform` in the linux X86-64 machine, you can copy `linux_amd64` binary into terraform plugins directory:

```
cp bin/terraform-provider-confluent-kafka_v0.1.0 ${HOME}/.terraform.d/plugins/registry.terraform.io/hashicorp/confluent-kafka/0.1.0/linux_amd64
```

### 2. Start your first terraform plan

Create `main.tf` file:

```shell script
terraform {
  required_providers {
    confluent-kafka = {
      version = "0.1.0"
      source = "confluent-kafka"
    }
  }
}
```

Init

```shell script
terraform init
```

Plan

```shell script
terraform plan
```

Apply

```shell script
terraform apply
```

## 3. Resources supported

- Define the provider

```shell
provider "confluent-kafka" {
  alias = "confluent" # Alias of the provider
  bootstrap_servers = ["localhost:9093"] # List of Kafka bootstrap servers
  ca_cert           = "certs/ca.pem" # CA cert to connect to Kafka cluster
  client_cert       = "certs/cert.pem" # Client certs to connect to Kafka cluster
  client_key        = "certs/key.pem" # Client private key to connect to Kafka cluster
  skip_tls_verify   = true # Skip TlS verification
  username = "xxxx" # LDAP or SASL username to connect to Confluent or MDS
  password = "yyyy" # LDAP or SASL password to connect to Confluent or MDS
}
```

### 3.1 Topics

- Topic Example

```shell script
resource "kafka_topic" "example_topic" {
  cluster_id = "kafka-cluster-id" # Optional: If not, terraform will use first cluster in the cluster list
  name = "test-terraform-confluent-provider" # Topic name
  replication_factor = 3 # Replication factor
  partitions = 5 # The number of partition

  config = {
    "segment.ms" = "20000"
    "cleanup.policy" = "compact"
  }
  provider = confluent-kafka.confluent
}
```

### 3.2 Cluster role binding

- Will describe and bind the cluster role to principal (User or scope)

- Example:

```shell
resource "cluster_role_binding" "example_role_binding" {
  cluster_id = "kafka-cluster-id" # Optional: If not, terraform will use first cluster in the cluster list
  role = "UserAdmin" # Allow roles: "AuditAdmin", "ClusterAdmin", "DeveloperManage", "DeveloperRead", "DeveloperWrite", "Operator", "ResourceOwner", "SecurityAdmin", "SystemAdmin", "UserAdmin",
  principal = "User:wayarmy" # Allow convention: User:<user_name> or Group:<group_name>
  cluster_type = "Kafka" # Support 4 types of clusters: Kafka, SchemaRegistry, KSQL, Connect
  provider = confluent-kafka.confluent
}
```

### 3.3 Kafka topic RBAC

- Will describe and bind the resource role (Not Cluster role) to principal

- Example:

```shell
resource "kafka_topic_rbac" "example_topic_rbac" {
  principal = "User:wayarmy" # Allow convention: User:<user_name>, Group:<group_name>, User:CN=<domain>
  role = "ResourceOwner" # Allow roles: "DeveloperRead", "DeveloperWrite", "Operator", "ResourceOwner"
  resource_type = "Topic" # Allow only: Topic
  pattern_type = "PREFIXED" # Allow: PREFIXED and LITERAL
  name = "system-platform-" # The pattern contains in topic name
  cluster_id = "kafka-cluster-id" # Optional: If not, terraform will use first cluster in the cluster list
  provider = confluent-kafka.confluent
}

```

### 3.4 Schema Registry Subject RBAC

- Will describe and bind the subject acces roles to an user

- Example

```shell
resource "schema_registry_rbac" "example_role_binding_developerwrite_schema_subject" {
  cluster_id = "kafka-cluster-id"
  schema_registry_cluster_id = "schema-registry-cluster-name"
  role = "DeveloperWrite"
  principal = "User:wayarmy"
  name = "system-platform-"
  pattern_type = "PREFIXED"
  provider = confluent-kafka.confluent
}
```

### 3.5 Connector RBAC

- Will decribe and bind the connector access to a scope (user or another type of principal)

- Example

```shell
resource "connectors_rbac" "example_role_binding_developerwrite_connector" {
  cluster_id = "kafka-cluster-id"
  connect_cluster_id = "connector-cluster-name"
  role = "DeveloperRead"
  principal = "User:wayarmy"
  name = "system-platform-"
  pattern_type = "PREFIXED"
  provider = confluent-kafka.confluent
}
```

## 5. Contributing

- Clone this project

```shell
git clone git@gitlab.id.vin:sp/terraform-provider-confluent-kafka.git
```

- Build your personal environment terraform plugin for testing
```shell
./build.sh
```

- Run your test before push your code

```shell
go test
```

