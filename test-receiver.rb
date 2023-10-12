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

stdin.puts 'subscribe'
stdin.puts 'broadcast'
stdin.puts 'all'

stdin.puts 'subscribe'
stdin.puts 'queue'
stdin.puts 'q'

loop do
  line = stdout.gets
  puts line
end
