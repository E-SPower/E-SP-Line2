#!/bin/bash
# 从 VS Code 本地历史恢复整个 E-SP-Line2 项目所有文件到最新版本
# 用法: bash restore_all_from_history.sh
# 记 垃圾Deepseek删除了我的全部前端 臭肥鱼滚蛋吧

HISTORY_DIR="/home/typer/.vscode-server/data/User/History"
PROJECT_PREFIX="E-SP-Line2"
RESTORED=0
SKIPPED=0
FAILED=0

echo "=== 开始从 VS Code 本地历史恢复整个 E-SP-Line2 项目 ==="

# 遍历所有本地历史条目
for entry in "$HISTORY_DIR"/*/entries.json; do
    [ -f "$entry" ] || continue
    histdir=$(dirname "$entry")
    
    # 读取 resource 和 entries
    resource=$(python3 -c "import json;d=json.load(open('$entry'));print(d.get('resource',''))" 2>/dev/null)
    [ -z "$resource" ] && continue
    
    # 只处理 E-SP-Line2 下的文件（排除 E-SP-Line2-e2e-tests 和 ESPL2 旧路径）
    case "$resource" in
        *"$PROJECT_PREFIX"*)
            # 排除 e2e-tests 和旧路径
            case "$resource" in
                *"E-SP-Line2-e2e-tests"*|*"ESPL2"*) continue ;;
            esac
            
            # 提取相对路径 (E-SP-Line2/xxx)
            relpath=$(echo "$resource" | python3 -c "
import sys,urllib.parse
s=sys.stdin.read().strip()
s=urllib.parse.unquote(s)
idx=s.find('E-SP-Line2/')
if idx>=0:
    print(s[idx:])
else:
    print('')
" 2>/dev/null)
            [ -z "$relpath" ] && continue
            
            # 获取最新版本 id
            latest=$(python3 -c "import json;d=json.load(open('$entry'));print(d['entries'][-1]['id'])" 2>/dev/null)
            [ -z "$latest" ] && continue
            
            srcfile="$histdir/$latest"
            dstfile="$relpath"
            
            if [ -f "$srcfile" ]; then
                # 确保目标目录存在
                mkdir -p "$(dirname "$dstfile")"
                cp "$srcfile" "$dstfile"
                echo "恢复: $dstfile (← $latest)"
                RESTORED=$((RESTORED+1))
            else
                echo "跳过(源文件不存在): $dstfile"
                SKIPPED=$((SKIPPED+1))
            fi
            ;;
    esac
done

echo ""
echo "=== 恢复完成 ==="
echo "恢复文件数: $RESTORED"
echo "跳过文件数: $SKIPPED"
echo "失败文件数: $FAILED"
