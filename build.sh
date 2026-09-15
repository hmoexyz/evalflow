# 当前目录
CURRENT_DIR=$(dirname $(readlink -f $0))

# 构建后端
cd "${CURRENT_DIR}/backend"
go build

# 构建前端
cd "${CURRENT_DIR}/frontend"
npm install
npm run build