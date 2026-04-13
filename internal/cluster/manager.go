package cluster

import (
	"time"

	"vault-kv/internal/config"
)

func Init(cfg *config.Config) Cluster {
	clusterBuilder := NewClusterBuilder()

	if cfg.ClusterReplicas == 0 {
		cfg.ClusterReplicas = 50
	}
	clusterBuilder.SetReplicas(cfg.ClusterReplicas)

	serverTimeout := 5 * time.Second
	if cfg.ServerTimeout != "" {
		if t, err := time.ParseDuration(cfg.ServerTimeout); err == nil {
			serverTimeout = t
		}
	}
	clusterBuilder.SetServerTimeout(serverTimeout)

	clusterBuilder.SetNodes(cfg.ClusterNodes)
	clusterBuilder.SetHttpClient(cfg)

	return clusterBuilder.Build()
}
