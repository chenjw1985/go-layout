
1. 启动Apollo配置中心
```bash
docker-compose up
```

2. 输入用户名apollo，密码admin后登录Apollo Portal
```bash
http://localhost:8070
```

3. 创建 go-layout 项目，并创建application.yaml的namespace

4. 配置Apollo Config service的环境变量
```bash
vim ~/.bash_profile

export APOLLO_APPID=go-layout
export APOLLO_CLUSTER=dev
export APOLLO_ENDPOINT=http://localhost:8080
export APOLLO_NAMESPACE=application.yaml
export APOLLO_SECRET=ad75b33c77ae4b9c9626d969c44f41ee
```
