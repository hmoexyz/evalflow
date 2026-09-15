FROM ubuntu:latest

# 安装系统依赖
RUN apt-get update && \
    apt-get install -y ca-certificates && \
    update-ca-certificates

# 复制可执行文件
COPY evalflow /opt/backend/evalflow
RUN chmod +x /opt/backend/evalflow

# 设置启动命令
CMD ["/opt/backend/evalflow"]