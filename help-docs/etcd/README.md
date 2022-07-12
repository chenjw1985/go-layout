1. Run ETCD on local
```bash
docker run -d --name etcd \
  -e ALLOW_NONE_AUTHENTICATION=yes \
  -e ETCD_ADVERTISE_CLIENT_URLS=http://etcd:2379 \
  -p 2379:2379 \
  -p 2380:2380 \
  bitnami/etcd:latest
```