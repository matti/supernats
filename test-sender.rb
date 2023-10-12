$stdout.sync = true
require 'open3'

url = ARGV[0] || raise('no url')

stdin, stdout, stderr, thread = Open3.popen3({},
                                             'supernats', url)

Thread.new do
  thread.join
  puts 'supernats died'
  exit
end

stderr_thr = Thread.new do
  while logline = stderr.gets
    warn logline
  end
end

counter = 0
loop do
  counter += 1
  stdin.puts 'publish'
  stdin.puts 'all'
  stdin.puts counter.to_s

  stdin.puts 'publish'
  stdin.puts 'q'
  stdin.puts counter.to_s

  puts counter
  sleep 1
rescue Errno::EPIPE
  sleep 1
  retry
end

# line = stdout.gets
# exitstatus = thread&.value&.exitstatus
# if exitstatus == 0
#   return [true, line]
# else
#   warn "subito exit: #{exitstatus}"
#   return [false, nil]
# end
