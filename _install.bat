@echo off
rem run this script as admin

if not exist log1c.exe (
    echo Build the example before installing by running "go build"
    goto :exit
)

sc create log1c binpath= "%CD%\log1c.exe" start= auto DisplayName= "log1c"
sc description log1c "log1c"
sc start log1c
sc query log1c

echo Check log1c.exe

:exit
