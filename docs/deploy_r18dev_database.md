# 更新R18.dev数据库实例流程

## 下载最新数据库转储

发布页：https://r18.dev/dumps

或者直接下载最新转储文件：https://r18.dev/dumps/latest

## 创建数据库

在PostgreSQL中创建新的数据库，所有者设置为`postgres_user`，编码选择`UTF8`

## 写入转储

在进行这一步之前，将需要写入的转储文件放置到`homserver0`的以下路径`/home/wangqi/services/.compose_data/jav/temp`

```shell
# 找到目标容器ID
sudo docker ps | grep jav-postgre

# 进入容器
sudo docker exec -it ${填入容器ID} bash

# 执行写入 ( postgres_user -> 数据库用户名 | r18dotdev_20250603 -> 目标数据库名 | r18dotdev_dump_2025-06-03.sql -> 要写入的转储文件名 )
psql -U postgres_user -d r18dotdev_20250603 -f /temp/r18dotdev_dump_2025-06-03.sql

# 退出容器
exit
```

## 检查是否成功

使用`Navicat Premium Lite 17`打开PostgreSQL数据库连接，`Navicat Premium 16`不行，打不开新版PostgreSQL数据库