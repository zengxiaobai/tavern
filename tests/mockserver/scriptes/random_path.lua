local chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
local chars_len = #chars

function init(args)
    local base_seed = 20050520
    math.randomseed(base_seed)
end

function random_string(len)
    local t = {}
    for i = 1, len do
        local idx = math.random(1, chars_len)
        t[i] = string.sub(chars, idx, idx)
    end
    return table.concat(t)
end

request = function()
    local id = random_string(22)
    local path = "/path/" .. id
    return wrk.format("GET", path)
end
