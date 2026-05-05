# 定义目标URL
$url1 = "http://localhost:9999/api?key=Jack"
$url2 = "http://localhost:9999/api?key=Tom"
# 循环100次
1..100 | ForEach-Object {
    Start-Process curl.exe -ArgumentList "$url1" -NoNewWindow
    Start-Process curl.exe -ArgumentList "$url2" -NoNewWindow
}

Write-Host "finished"