Output assets/demo.gif
Output assets/demo.mp4

Require ./bin/linux-mcp
Require jq

Set FontSize 18
Set Width 1100
Set Height 700
Set Padding 30
Set Margin 60
Set MarginFill "#11111b"
Set WindowBar Colorful
Set WindowBarSize 36
Set BorderRadius 12
Set Theme "Catppuccin Mocha"
Set Framerate 15
Set PlaybackSpeed 1.25
Set CursorBlink false
Set TypingSpeed 25ms

Hide
Type "clear"
Enter
Sleep 1s
Show

Type "./scripts/demo.sh get_system_info"
Enter
Wait+Screen /procs/
Sleep 2s

Type "clear"
Enter
Sleep 500ms

Type "./scripts/demo.sh get_cpu_info"
Enter
Wait+Screen /core_count/
Sleep 2s

Type "clear"
Enter
Sleep 500ms

Type "./scripts/demo.sh get_memory_info"
Enter
Wait+Screen /total/
Sleep 3s
