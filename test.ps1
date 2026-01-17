# 定义目标 URL
$url = "http://localhost:9001/api?key=Jack"

# 循环 3 次
1..100 | ForEach-Object {
    # Start-Process 类似于 start，独立启动 curl.exe 进程
    # -NoNewWindow 表示不弹出新窗口
    Start-Process curl.exe -ArgumentList "$url" -NoNewWindow
}

Write-Host "finished"