# OpenKanban Integration Test
#
# This tape validates core functionality works end-to-end.
# Run with: VHS_NO_SANDBOX=true vhs tests/integration.tape
#
# Requires: vhs, git, openkanban binary in PATH or current dir

Output tests/integration.gif

Set Shell "bash"
Set FontSize 14
Set Width 1000
Set Height 600
Set Padding 10

# Setup: Create temp directory with git repo
Hide
Type "cd $(mktemp -d)"
Enter
Sleep 100ms
Type "git init"
Enter
Sleep 100ms
Type "git commit --allow-empty -m 'init'"
Enter
Sleep 100ms
Show

# Test 1: Version command works
Type "openkanban version"
Enter
Sleep 500ms

# Test 2: Create a new project
Type "openkanban new 'Integration Test'"
Enter
Sleep 500ms

# Test 3: Launch TUI
Type "openkanban"
Enter
Sleep 1s

# Test 4: Create a ticket
Type "n"
Sleep 500ms
Type "Test ticket from CI"
Tab
Sleep 200ms
Type "This validates the TUI works"
Ctrl+d
Sleep 1s

# Test 5: Move ticket to in-progress
Type " "
Sleep 500ms

# Test 6: Clean exit
Type "q"
Sleep 500ms
