# DBaaS in 2024: Which PostgreSQL operator for Kubernetes to select for your platform?

1. part - Requirements https://medium.com/@davidpech_39825/dbaas-in-2024-which-postgresql-operator-for-kubernetes-to-select-for-your-platform-51cf4d5dec4a


## ArgoCD in the cluster

```
export GIT_REPO=https://github.com/depeche-io/dbaas-postgresql.git/argocd-gitops
export GIT_TOKEN=XXX # retrieve from your GitHub account

# after installation there should be these instructions:
# INFO argocd initialized. password: XXX 
kubectl port-forward -n argocd svc/argocd-server 8080:80

# so just admin / previous password here:
open http://localhost:8080/
```