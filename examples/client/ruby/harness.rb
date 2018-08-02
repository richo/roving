require 'rubygems'
require 'bundler/setup'

require 'afl'

def byte
  $stdin.read(1)
end

def c
  r if byte == 'r'
end

def r
  s if byte == 's'
end

def s
  h if byte == 'h'
end

def h
  raise "Crashed"
end

unless ENV['NO_AFL']
  AFL.init
end

AFL.with_exceptions_as_crashes do
  c if byte == 'c'
  exit!(0)
end
