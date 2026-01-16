@echo off
echo 编译SheetDiff...
go build -o SheetDiff.exe
if errorlevel 1 (
    echo 编译失败！
    pause
    exit /b 1
)
echo 编译成功！
pause