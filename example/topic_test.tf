resource "cluster_role_binding" "example_role_binding" {
  cluster_id = "cluster-1"
  role = "UserAdmin"
  principal = "User:user-test"
  cluster_type = "Kafka"
  provider = confluent-kafka.confluent
}

resource "cluster_role_binding" "example_role_binding_operator" {
  cluster_id = "cluster-1"
  role = "Operator"
  principal = "User:user-test"
  cluster_type = "Kafka"
  provider = confluent-kafka.confluent
}

resource "kafka_topic_rbac" "example_topic_rbac" {
  principal = "User:user-test"
  role = "ResourceOwner"
  resource_type = "Topic"
  pattern_type = "PREFIXED"
  name = "test-"
  cluster_id = "cluster-1"
  provider = confluent-kafka.confluent
}

resource "kafka_topic_rbac" "example_cn_topic_rbac" {
  principal = "User:CN=common-name.example.com"
  role = "ResourceOwner"
  resource_type = "Topic"
  pattern_type = "LITERAL"
  name = "test-terraform-confluent-provider"
  cluster_id = "cluster-1"
  provider = confluent-kafka.confluent
}

resource "kafka_topic" "example_topic" {
  cluster_id = "cluster-1"
  name = "test-terraform-confluent-provider"
  replication_factor = 3
  partitions = 5

  config = {
    "segment.ms" = "20000"
    "cleanup.policy" = "compact"
    "retention.ms" = "300000"
  }
  provider = confluent-kafka.confluent
}