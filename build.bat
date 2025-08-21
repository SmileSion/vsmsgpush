@echo off
rem ------------------------- 配置部分 -------------------------
set OUTPUT_DIR=bin

rem 需要编译的 main.go 清单（格式：子目录名=main.go 路径）
rem 左边是生成的文件名，右边是 main.go 的路径
set TARGETS=VxMsgPush=cmd\Push\main.go VxApi=cmd\Api\main.go TokenService=cmd\Token\main.go

rem ------------------------- 检查并创建输出目录 -------------------------
if not exist %OUTPUT_DIR% (
    mkdir %OUTPUT_DIR%
)

rem ------------------------- 编译不同平台 -------------------------
set GOOS=linux
set GOARCH=amd64

for %%i in (%TARGETS%) do (
    for /f "tokens=1,2 delims==" %%a in ("%%i") do (
        echo 正在编译 %%a ...
        go build -o %OUTPUT_DIR%\%%a-linux-amd64 %%b
    )
)

echo 所有平台的可执行文件已经生成。
pause
