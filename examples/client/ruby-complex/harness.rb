def byte
  $stdin.read(1)
end

def entry_point
  if (byte == "4")
    f0
  else
    f1
  end
end

def f0
  if (byte == "e")
    f2
  else
    f3
  end
end

def f1
  if (byte == "1")
    f4
  else
    f5
  end
end

def f2
  if (byte == "4")
    f6
  else
    f7
  end
end

def f3
  if (byte == "0")
    f8
  else
    f9
  end
end

def f4
  if (byte == "0")
    f10
  else
    f11
  end
end

def f5
  if (byte == "3")
    f12
  else
    f13
  end
end

def f6
  if (byte == "3")
    f14
  else
    f15
  end
end

def f7
  if (byte == "0")
    f16
  else
    f17
  end
end

def f8
  if (byte == "4")
    f18
  else
    f19
  end
end

def f9
  if (byte == "4")
    f20
  else
    f21
  end
end

def f10
  if (byte == "9")
    f22
  else
    f23
  end
end

def f11
  if (byte == "c")
    f24
  else
    f25
  end
end

def f12
  if (byte == "6")
    f26
  else
    f27
  end
end

def f13
  if (byte == "b")
    f28
  else
    f29
  end
end

def f14
  if (byte == "9")
    raise(RuntimeError)
  else
    "not throwing!"
  end
end

def f15
  if (byte == "e")
    raise(RuntimeError)
  else
    "not throwing!"
  end
end

def f16
  if (byte == "7")
    raise(RuntimeError)
  else
    "not throwing!"
  end
end

def f17
  if (byte == "0")
    raise(RuntimeError)
  else
    "not throwing!"
  end
end

def f18
  if (byte == "9")
    raise(RuntimeError)
  else
    "not throwing!"
  end
end

def f19
  if (byte == "3")
    raise(RuntimeError)
  else
    "not throwing!"
  end
end

def f20
  if (byte == "b")
    raise(RuntimeError)
  else
    "not throwing!"
  end
end

def f21
  if (byte == "9")
    raise(RuntimeError)
  else
    "not throwing!"
  end
end

def f22
  if (byte == "e")
    raise(RuntimeError)
  else
    "not throwing!"
  end
end

def f23
  if (byte == "4")
    raise(RuntimeError)
  else
    "not throwing!"
  end
end

def f24
  if (byte == "4")
    raise(RuntimeError)
  else
    "not throwing!"
  end
end

def f25
  if (byte == "c")
    raise(RuntimeError)
  else
    "not throwing!"
  end
end

def f26
  if (byte == "7")
    raise(RuntimeError)
  else
    "not throwing!"
  end
end

def f27
  if (byte == "c")
    raise(RuntimeError)
  else
    "not throwing!"
  end
end

def f28
  if (byte == "6")
    raise(RuntimeError)
  else
    "not throwing!"
  end
end

def f29
  if (byte == "d")
    raise(RuntimeError)
  else
    "not throwing!"
  end
end

require 'afl'

unless ENV['NO_AFL']
  AFL.init
end

AFL.with_exceptions_as_crashes do
  entry_point()
  exit!(0)
end

