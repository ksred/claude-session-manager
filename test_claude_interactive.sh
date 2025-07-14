#!/bin/bash

echo "Testing Claude CLI in interactive mode..."
echo "=========================="

# Try running claude with a pty to simulate interactive mode
echo "Testing with script command to simulate TTY:"
script -q /dev/null bash -c 'echo -e "hello\nexit" | timeout 5s claude 2>&1 | cat -v' | head -50

echo ""
echo "=========================="
echo "Testing with expect if available:"
if command -v expect &> /dev/null; then
    expect -c '
        spawn claude
        set timeout 5
        expect {
            timeout { puts "TIMEOUT waiting for prompt" }
            eof { puts "EOF reached" }
            -re ".*" { 
                puts "GOT OUTPUT: $expect_out(buffer)"
                send "hello\r"
                expect -re ".*"
                puts "RESPONSE: $expect_out(buffer)"
            }
        }
    '
else
    echo "expect not available"
fi