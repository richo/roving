# NOTE: requires installing gems:
#
# * `parser` - https://github.com/whitequark/parser
# * `unparser` - https://github.com/mbj/unparser
require 'securerandom'
require 'parser/current'
require 'unparser'

# Codegen a big Ruby program of the form:
#
# def x1
#   if byte == 'a'
#     x2
#   else
#     x3
#   end
# end
#
# def x2
#   if byte == 'b'
#     x4
#   else
#     x5
#   end
# end
#
# def x3
#   if byte == 'c'
#     raise RuntimeError
#   else
#     "not throwing"
#   end
# end
#
# # etc....

# Usage:
#
# ```
# ruby generate_def_code.rb | pbcopy
# ```
#
# Then paste the output into a harness.rb file, and start
# fuzzing it!

# Returns a node that represents the code:
#
# def $NAME
#   if byte == $RANDOM_CHAR
#     # Run true_node
#   else
#     # Run false_node
#   end
# end
def def_node(name, true_node, false_node)
  char = SecureRandom.hex[0]
  Parser::AST::Node.new(
    :def,
    [
      name,
      Parser::AST::Node.new(:args, []),
      if_byte_eq_node(
        char, 
        true_node,
        false_node,
      )
    ]
  )
end

# Returns a node that represents the code:
#
# $F_NAME()
def call_f_node(f_name)
  Parser::AST::Node.new(
    :send,
    [nil, f_name]
  )
end

# Returns a node that represents the code:
#
# raise RuntimeError
def raise_node
  Parser::AST::Node.new(
    :send,
    [
      nil,
      :raise,
      Parser::AST::Node.new(
        :const,
        [nil, :RuntimeError],
      )
    ]
  )
end

# Returns a node that represents the code:
#
# "not throwing!"
def happy_node
  Parser::AST::Node.new(
    :str,
    ["not throwing!"]
  )
end

# Returns a node that represents the code:
#
# if byte == $CHAR
#   # Run true_node
# else
#   # Run false_node
# end
def if_byte_eq_node(char, true_node, false_node)
  Parser::AST::Node.new(
    :if,
    [
      Parser::AST::Node.new(
        :send,
        [
          Parser::AST::Node.new(
            :send,
            [nil, :byte]
          ),
          :==, 
          Parser::AST::Node.new(
            :str,
            [char]
          )
        ]
      ),
      true_node,
      false_node,
    ]
  )
end

# We name functions `f0`, `f1`, `f2` etc. We are sorry for the global var
$next_f_number = 0

# Returns an array of nodes that represent function definitions. All
# functions have names of the form `x123456`.
def def_tree_of_depth(depth, def_nodes, f_names_to_define)
  if depth > 0
    new_def_nodes, new_f_names_to_define = f_names_to_define.map do |f_name|
      if depth > 1
        true_f_name = :"f#{$next_f_number}"
        $next_f_number += 1
        false_f_name = :"f#{$next_f_number}"
        $next_f_number += 1

        true_node = call_f_node(true_f_name)
        false_node = call_f_node(false_f_name)

        new_def_node = def_node(f_name, true_node, false_node)
        [new_def_node, [true_f_name, false_f_name]]
      else
        true_node = raise_node
        false_node = happy_node

        new_def_node = def_node(f_name, true_node, false_node)
        [new_def_node, []]
      end
    end.transpose

    new_f_names_to_define.flatten!
    def_tree_of_depth(
      depth - 1,
      def_nodes + new_def_nodes,
      new_f_names_to_define,
    )
  else
    [0, def_nodes, []]
  end
end

def generate_def_nodes(depth)
  def_tree_of_depth(depth, [], [:entry_point])[1]
end

nodes = generate_def_nodes(5)

# Print afl-ruby boilerplate
puts %{
def byte
  $stdin.read(1)
end
}
nodes.each do |node|
  puts ''
  puts Unparser.unparse(node)
end

# Print more afl-ruby boilerplate
puts %{
require 'afl'

unless ENV['NO_AFL']
  AFL.init
end

AFL.with_exceptions_as_crashes do
  entry_point()
  exit!(0)
end
}
