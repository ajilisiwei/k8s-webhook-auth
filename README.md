## webhook 方式认证

### 概述

Webhook 身份认证是一种用来验证持有者令牌的回调机制。

    - --authentication-token-webhook-config-file 指向一个配置文件， 其中描述如何访问远程的 Webhook 服务。
    - --authentication-token-webhook-cache-ttl 用来设定身份认证决定的缓存时间。 默认时长为 2 分钟。

### 创建 webhook 服务

webhook 服务的请求体和返回体需满足 k8s 的 TokenView 规范定义：

1. 请求规范

   ```sh
   {
     "apiVersion": "authentication.k8s.io/v1",
     "kind": "TokenReview",
     "spec": {
      # 发送到 API 服务器的不透明持有者令牌
       "token": "014fbff9a07c...",
      
       # 提供令牌的服务器的受众标识符的可选列表。
       # 受众感知令牌验证器（例如，OIDC 令牌验证器）
       # 应验证令牌是否针对此列表中的至少一个受众，
       # 并返回此列表与响应状态中令牌的有效受众的交集。
       # 这确保了令牌对于向其提供给的服务器进行身份验证是有效的。
       # 如果未提供受众，则应验证令牌以向 Kubernetes API 服务器进行身份验证。
       "audiences": ["https://myserver.example.com", "https://myserver.internal.example.com"]
     }
   }
   ```
2. 响应规范

   ```sh
   #  请求成功返回格式
   {
     "apiVersion": "authentication.k8s.io/v1",
     "kind": "TokenReview",
     "status": {
       "authenticated": true,
       "user": {
         # 必要
         "username": "janedoe@example.com",
         # 可选
         "uid": "42",
         # 可选的组成员身份
         "groups": ["developers", "qa"],
         # 认证者提供的可选附加信息。
         # 此字段不可包含机密数据，因为这类数据可能被记录在日志或 API 对象中，
         # 并且可能传递给 admission webhook。
         "extra": {
           "extrafield1": [
             "extravalue1",
             "extravalue2"
           ]
         }
       },
       # 验证器可以返回的、可选的用户感知令牌列表，
       # 包含令牌对其有效的、包含于 `spec.audiences` 列表中的受众。
       # 如果省略，则认为该令牌可用于对 Kubernetes API 服务器进行身份验证。
       "audiences": ["https://myserver.example.com"]
     }
   }

   # 请求不成功返回格式
   {
     "apiVersion": "authentication.k8s.io/v1",
     "kind": "TokenReview",
     "status": {
       "authenticated": false,
       # 可选地包括有关身份验证失败原因的详细信息。
       # 如果没有提供错误信息，API 将返回一个通用的 Unauthorized 消息。
       # 当 authenticated=true 时，error 字段被忽略。
       "error": "Credentials are expired"
     }
   }
   ```

说明：
   Kubernetes API 服务器默认发送 authentication.k8s.io/v1beta1 令牌以实现向后兼容性。 要选择接收 authentication.k8s.io/v1 令牌认证，API 服务器必须带着参数 --authentication-token-webhook-version=v1 启动。

### 为 kube-apiserver 添加 webhook 认证配置

以下为 kube-apiserver 的 webhook 的 kubeconfig 文件的配置格式。
```sh
# Kubernetes API 版本
apiVersion: v1
# API 对象类别
kind: Config
# clusters 指代远程服务
clusters:
  - name: name-of-remote-authn-service
    cluster:
      certificate-authority: /path/to/ca.pem         # 用来验证远程服务的 CA
      server: https://authn.example.com/authenticate # 要查询的远程服务 URL。生产环境中建议使用 'https'。

# users 指代 API 服务的 Webhook 配置
users:
  - name: name-of-api-server
    user:
      client-certificate: /path/to/cert.pem # Webhook 插件要使用的证书
      client-key: /path/to/key.pem          # 与证书匹配的密钥

# kubeconfig 文件需要一个上下文（Context），此上下文用于本 API 服务器
current-context: webhook
contexts:
- context:
    cluster: name-of-remote-authn-service
    user: name-of-api-server
  name: webhook
```

### 实验案例

1. 启动 webhook 认证服务

```sh
# 机器IP：192.168.56.3
wei@wei1:~$ cd k8s-webhook-auth
wei@wei1:~$ make
wei@wei1:~$ ./k8s-webhook-auth
2022/09/25 16:21:03 Listen on port: 3000
```

2. 修改 kube-apiserver 启动配置

```sh
wei@wei1:~/k8s-webhook-auth$ mkdir -p /etc/config
wei@wei1:~k8s-webhook-auth$ cp webhook-config.yaml /etc/config

# vim /etc/kubernetes/manifests/kube-apiserver.yaml
# 增加启动参数 --authentication-token-webhook-config-file 
- --authentication-token-webhook-config-file=/etc/config/webhook-config.yaml

# 增加 volumeMounts 配置
- name: webhook-config
  mountPath: /etc/config
  readOnly: true

# 增加 volumes 配置
- hostPath:
    path: /etc/config
    type: DirectoryOrCreate
  name: webhook-config
```
保存，待 kube-apiserver 重启成功。

3. 增加 ~/.kube/config 配置
```sh
# users 增加用配置
- name: wei2
  user:
    token: wei2-token
```

4. 测试
```sh
wei@wei0:/etc/config$ kubectl get po --user=wei2
Error from server (Forbidden): pods is forbidden: User "wei2" cannot list resource "pods" in API group "" in the namespace "default"
```
显示用户名 wei2 已经通过 webhook 认证，只是还没为 wei2 授权，所以还无法请求资源

```sh
# 可尝试将 ~/.kube/config 的 token 修改为错误的token，查看认证失败的情况
wei@wei0:/etc/config$ kubectl get po --user=wei2
error: You must be logged in to the server (Unauthorized)
```

### 附录

- [Webhook 令牌身份认证](https://kubernetes.io/zh-cn/docs/reference/access-authn-authz/authentication/#webhook-token-authentication)