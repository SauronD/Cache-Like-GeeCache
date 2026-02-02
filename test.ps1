# 定义目标 URL
$url1 = "http://localhost:9999/api?key=Jack"
$url2 = "http://localhost:9999/api?key=Tom"
# 循环 3 次
1..100 | ForEach-Object {
    # Start-Process 类似于 start，独立启动 curl.exe 进程
    # -NoNewWindow 表示不弹出新窗口
    Start-Process curl.exe -ArgumentList "$url1" -NoNewWindow
    Start-Process curl.exe -ArgumentList "$url2" -NoNewWindow
}

Write-Host "finished"