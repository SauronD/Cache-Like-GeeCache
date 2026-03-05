#!/bin/bash

# 遇到错误即退出，并在脚本结束时（包括按 Ctrl+C）清理所有后台进程
trap "rm -f likecache-server; kill 0" EXIT

echo ">>> 🔨 1. 正在编译项目..."
go build -o likecache-server main.go

echo ">>> 🌐 2. 启动 Registry 注册中心 (端口 9000)..."
./likecache-server -r -p 9001 &
sleep 1 # 等待注册中心启动完成

echo ">>> 📦 3. 启动 缓存节点 1 (端口 8001)..."
./likecache-server -p 9002 &

echo ">>> 📦 4. 启动 缓存节点 2 (端口 8002)..."
./likecache-server -p 9003 &

echo ">>> 🚪 5. 启动 缓存节点 3 + API 网关 (端口 8003, API 9999)..."
./likecache-server -p 9004 -api &

echo ">>> ⏳ 6. 等待节点互相注册与心跳哈希环同步 (等待 10 秒)..."
sleep 10

echo -e "\n============================================="
echo "🧪 测试 1：基础功能与跨节点路由 (正常获取)"
echo "============================================="
echo "请求 scores 组的 Tom:"
curl "http://localhost:9999/api?group=scores&key=Tom"
echo -e "\n请求 age 组的 Jack:"
curl "http://localhost:9999/api?group=age&key=Jack"
echo ""

echo -e "\n============================================="
echo "🧪 测试 2：布隆过滤器防穿透拦截 (查不存在的 Key)"
echo "预期：不会触发底层 DB 查询，直接返回 bloom filter 拦截错误"
echo "============================================="
curl "http://localhost:9999/api?group=scores&key=Ghost"
echo ""

echo -e "\n============================================="
echo "🧪 测试 3：高并发防击穿 (Singleflight)"
echo "预期：瞬间发起 10 个并发请求，但控制台只会打印 1 次底层 DB 获取日志"
echo "============================================="

# 定义一个数组用来存 curl 的进程 ID
pids=() 

for i in {1..10}; do
    curl -s "http://localhost:9999/api?group=scores&key=Sam" &
    pids+=($!) # 把刚刚启动的 curl 的 PID 存起来
done

# 🌟 修复点：只等待这 10 个 curl 进程，不管服务器进程
wait "${pids[@]}" 

echo -e "\n(请观察服务端的终端日志，是否只触发了一次查库)"

echo -e "\n============================================="
echo "🧪 测试 4：热点数据缓存命中 (HotCache)"
echo "预期：连续请求同一个不归本机管辖的 Key，必定能触发10%概率，随后实现 [HotCache] hit"
echo "============================================="
# 循环 30 次，必中 10% 概率
for i in {1..30}; do
    curl -s "http://localhost:9999/api?group=age&key=Sam" > /dev/null
done
echo ">>> 30 次高频请求已发送！请查看服务端日志是否出现 [HotCache] hit ！"

echo -e "\n>>> 🎉 所有测试脚本执行完毕！(按 Ctrl+C 关闭整个集群)"
wait