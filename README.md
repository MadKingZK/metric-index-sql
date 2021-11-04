# Metric-Index-Sql

## 项目介绍
**接收Prometheus或vminsert的RemoteWrite请求，去重后写入Kafka中**

```go
Metric = MetricName + Labels
Labels = []{Label.Name=Label.Value}
```

- **接收RemoteWrite，提取每个Metric的labels数组，组装成metric指标名称字符串**
- **将Metric存入Gocache、Redis和Kafka中**
    - Gocache提供本地缓存Metric，减轻reids负担，提供Exist判断，过期时间可配置
    - Redis缓存Metric，提供Exist判断，过期时间可配置
    - Kafka存储去重后的Metric，下游worker节点会消费到mysql中


## 项目结构

- routers 路由层，URL配置，项目接口入口
- controller 控制器层，验证提交的数据，将验证完成的数据传递给 service
- service 业务层，只完成业务逻辑的开发，不进行操作数据库
- dao 数据库层，操作数据的CURD

## 项目组件

- Viper 配置管理，监听配置，自动加载更新
- Zap 日志管理
- patrickmn/go-cache 本地缓存
- go-redis/redis Redis驱动
- go-sql-driver/mysql Mysql驱动
- jmoiron/sqlx sql扩展，简化数据库操作
- Shopify/sarama Kafka驱动
- bsm/sarama-cluster Kafka集群驱动
- gin-contrib/pprof Gin性能分析
- air 能够实时监听项目的代码文件，在代码发生变更之后自动重新编译并执行

## 项目编译

使用 golangci-lint和golint做代码规范检测，推荐使用golint

```bash
bash build_linux.sh
```

## 项目配置

- 环境变量配置

    目前只区分开发环境和线上环境，通过配置环境变量`export GO_ENV=dev`或`export GO_ENV=prod`

- 配置文件

```yaml
app:
  name: "metric-index"
  mode: "prod"
  role_type: "consummer"
  port: 7001

log:
  level: "info"
  filename: "logs/metric-index.log"
  max_size: 200
  max_age: 30
  max_backups: 7

mysql:
  host: 127.0.0.1
  port: 3306
  user: root
  password: mypassword
  db_name: "metric-index"
  max_open_conns: 200
  max_idel_conns: 50

redis:
  host: "127.0.0.1"
  port: 6379
  password: ""
  db: 0
  pool_size: 3000

metric_store:
  cache:
    isexpire: true
    expire: 604800
    dist_interval: 345600
    default_expire: 1800
    cleanup_interval: 60
    worker_num: 10
    flush_lens: 1000
    fulsh_interval: 3
  producer:
    hosts:
      - kafka-001:9092
      - kafka-002:9092
      - kafka-003:9092
    topic: "metric-index"
  consummer:
    hosts:
      - kafka-001:9092
      - kafka-002:9092
      - kafka-003:9092
    topics:
      - "metric-index"
    group_id: metric-index-worker-v1
    work_num: 8
    offset_type: newest

remote:
  write:
    url: "http://127.0.0.1:8480/insert/1/prometheus/api/v1/write"
    content_type: "application/x-protobuf"
  send:
    url: "http://127.0.0.1:4242/insert/1/opentsdb/api/put"
    content_type: "application/json"
```

## 项目启动

#### 测试环境，推荐使用Air启动
- 配置air配置文件，air能够实时监听项目的代码文件，在代码发生变更之后自动重新编译并执行，大大提高gin框架项目的开发效率
  - [Air实时加载](http://www.topgoer.cn/docs/ginkuangjia/ginairshishijiazai)
  - 根据上面教程安装air、配置.air.conf，需要修改full_bin，添加开发环境的环境变量参数
    ```bash
    full_bin = "export GO_ENV=dev; ./tmp/main"
    ```
  - 启动项目
    ```bash
    air -c .air.conf
    ```

- goland运行

```
项目配置（Edit Configurations） → Configuration → Environment → 添加：GO_ENV=dev
```

- 本地开发环境二进制启动(MAC)
    - 配置环境变量

    ```bash
    sudo echo 'export GO_ENV=dev' >> ~/.zshrc
    ```
    - config目录中创建配置文件：dev.yml
    - 执行脚本build_linux.sh打包编译
    - 启动项目：./metric-index


#### 线上环境
- 优化系统配置
    ```bash
    # 注释/etc/security/limits.d/20-nproc.conf内所有配置
    > sed -i 's/^[^#]/#&/g' /etc/security/limits.d/20-nproc.conf

    # 配置/etc/security/limits.d/limit.conf，重新登录中端使之生效
    > cat /etc/security/limits.d/limits.conf | egrep -v '^#|^$'
    root soft nofile 1024000
    root hard nofile 1024000
    * soft nofile 1024000
    * hard nofile 1024000
    ```

- 优化supervisor配置
    ```bash
    # 修改文件/etc/supervisord.conf中下面配置项，重启supervisor
    minfds=1024000
    minprocs=1024000
    ```
  
- 配置supervisor项目启动文件
    ```bash
    > cat /etc/supervisord.d/metric-index.conf
    [program:metric-index]
    command=/opt/metric-index/metric-index
    directory=/opt/metric-index/
    user=root
    environment=GO_ENV="prod"
    # ,GOMAXPROCS=7
    stderr_logfile=/var/log/supervisor/metric-index-err.log
    stdout_logfile=/var/log/supervisor/metric-index-info.log
    autostart=true
    autorestart=true
    startsecs=3
    ```

- 配置环境变量

    ```bash
    sudo echo 'export GO_ENV=prod' >> ~/.zshrc
    ```
    - config目录中创建配置文件：prod.yml
    - 执行脚本build_linux.sh打包编译，二进制文件metric-index发布到/opt/metric-index/
    - 启动项目：`supervisorctl update`、`supervisorctl start metric-index`
 

## 性能分析

- 项目配置了gin-contrib/pprof，可以通过pprof工具进行性能分析，接口为`/debug/pprof/`
- mac安装pprof
    - 安装 graphviz，支持打开svg文件
        ```bash
        brew install graphviz
        ```
  
    - 安装pprof工具
        ```bash
        go get github.com/gin-contrib/pprof
        ```
  
    - 测试pprof
        ```bash
        go tool pprof --help
        ```

- 执行pprof进行数据收集分析
    - cpu分析
        ```bash
        go tool pprof --seconds 60 http://[host]:[port]/debug/pprof/profile
        ```
  
    - memory分析
        ```bash
        go tool pprof --seconds 60 http://[host]:[port]/debug/pprof/heap
        ```

    - goroutine分析
        ```bash
        go tool pprof --seconds 60 http://[host]:[port]/debug/pprof/goroutine
        ```

- 查看分析结果
    - pprof指令执行完后，会提示生成的分析文件位置
    - 打开分析文件
    - 例如：`go tool pprof -http 127.0.0.1:port [pproffile path]`