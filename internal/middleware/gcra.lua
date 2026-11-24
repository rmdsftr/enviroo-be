local state_key, block_key = KEYS[1], KEYS[2]

local emission_ms = tonumber(ARGV[1])
local burst       = tonumber(ARGV[2])
local block_ms    = tonumber(ARGV[3])

local t = redis.call('TIME')
local now_ms = (tonumber(t[1]) * 1000) + math.floor(tonumber(t[2]) / 1000)

local block_ttl = redis.call('PTTL', block_key)
if block_ttl > 0 then
  return { 0, block_ttl }
end

local burst_ms = emission_ms * burst

local tat = tonumber(redis.call('GET', state_key))
if not tat or tat < now_ms then
  tat = now_ms
end

local new_tat  = tat + emission_ms
local allow_at = new_tat - burst_ms

if now_ms < allow_at then
  if block_ms > 0 then
    redis.call('SET', block_key, 1, 'PX', block_ms)
    return { 0, block_ms }
  end
  return { 0, 0 }
end

redis.call('SET', state_key, string.format('%.0f', new_tat),
           'PX', math.max(1, math.ceil(new_tat - now_ms)))

return { 1, 0 }
