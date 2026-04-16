-- create protocol
kcp_protocol = Proto("KCP", "KCP Protocol")

-- fields for kcp
conv = ProtoField.uint32("kcp.conv", "conv", base.DEC)
cmd = ProtoField.uint8("kcp.cmd", "cmd", base.DEC)
frg = ProtoField.uint8("kcp.frg", "frg", base.DEC)
wnd = ProtoField.uint16("kcp.wnd", "wnd", base.DEC)
ts = ProtoField.uint32("kcp.ts", "ts", base.DEC)
sn = ProtoField.uint32("kcp.sn", "sn", base.DEC)
una = ProtoField.uint32("kcp.una", "una", base.DEC)
len = ProtoField.uint32("kcp.len", "len", base.DEC)

kcp_protocol.fields = {conv, cmd, frg, wnd, ts, sn, una, len}

-- 【修复点 1】增强 get_cmd_name 函数，确保永远返回字符串
function get_cmd_name(cmd_val)
    if cmd_val == 81 then
        return "PSH"
    elseif cmd_val == 82 then
        return "ACK"
    elseif cmd_val == 83 then
        return "ASK"
    elseif cmd_val == 84 then
        return "TELL"
    elseif cmd_val == 51 then
        return "SIPKCP"
    else
        -- 如果是不认识的命令码，返回 "UNKNOWN (xx)" 格式，避免返回 nil
        return string.format("UNKNOWN (%d)", cmd_val)
    end
end

-- dissect each udp packet
function kcp_protocol.dissector(buffer, pinfo, tree)
    length = buffer:len()
    if length == 0 then
        return
    end

    -- 防止缓冲区读取越界的简单检查
    if buffer:len() < 32 then
        return
    end

    local offset_s = 8
    -- 确保有足够的字节读取
    if buffer:len() >= offset_s + 24 then
        local first_sn = buffer(offset_s + 12, 4):le_int()
        local first_len = buffer(offset_s + 20, 4):le_int()
        local cmd_val = buffer(offset_s + 4, 1):le_int()
        local first_cmd_name = get_cmd_name(cmd_val)

        -- 现在 first_cmd_name 绝不可能是 nil，string.format 安全
        local info = string.format("[%s] Sn=%d Kcplen=%d", first_cmd_name, first_sn, first_len)

        pinfo.cols.protocol = kcp_protocol.name

        -- 处理 info 列的显示逻辑
        local current_info = tostring(pinfo.cols.info)
        -- 替换 "Len" 为 "Udplen"
        local udp_info = string.gsub(current_info, "Len", "Udplen", 1)
        -- 插入 KCP 信息
        pinfo.cols.info = string.gsub(udp_info, " U", info .. " U", 1)
    end

    -- dissect multi kcp packet in udp
    local offset = 8
    while offset + 24 <= buffer:len() do
        local conv_buf = buffer(offset + 0, 4)
        local cmd_buf = buffer(offset + 4, 1)
        local wnd_buf = buffer(offset + 6, 2)
        local sn_buf = buffer(offset + 12, 4)
        local len_buf = buffer(offset + 20, 4)

        local cmd_val = cmd_buf:le_int()
        local cmd_name = get_cmd_name(cmd_val) -- 这里也绝不会再返回 nil
        local data_len = len_buf:le_int()

        --data_len 解析有问题，需要根据具体协议进行修改data_len的解析方式
        -- 边界检查：确保数据长度不会超出 buffer 剩余部分，防止死循环或崩溃
        --if offset + 24 + data_len > buffer:len() then
        -- 数据不完整，跳出循环或标记错误
        --     break
        -- end

        local tree_title =
        string.format(
                "KCP Protocol, %s, Sn: %d, Conv: %d, Wnd: %d, Len: %d",
                cmd_name,
                sn_buf:le_int(),
                conv_buf:le_int(),
                wnd_buf:le_int(),
                data_len
        )

        local subtree = tree:add(kcp_protocol, buffer(offset), tree_title)

        -- 添加字段到树
        subtree:add_le(conv, conv_buf)
        subtree:add_le(cmd, cmd_buf):append_text(" (" .. cmd_name .. ")")
        subtree:add_le(frg, buffer(offset + 5, 1))
        subtree:add_le(wnd, wnd_buf)
        subtree:add_le(ts, buffer(offset + 8, 4))
        subtree:add_le(sn, sn_buf)
        subtree:add_le(una, buffer(offset + 16, 4))
        subtree:add_le(len, data_len)

        -- 如果有负载数据，也可以尝试添加
        if data_len > 0 then
            -- subtree:add(buffer(offset + 24, data_len), "Data (" .. data_len .. " bytes)")
        end

        offset = offset + 24 + data_len
    end
end

-- register kcp dissector to udp
local udp_port = DissectorTable.get("udp.port")
-- 替换为你的 KCP 端口
udp_port:add(12992, kcp_protocol)