# VHS tape for the k8s-pod-autopsy README demo.
# Records real `autopsy` output against a live kind cluster with three
# deliberately-broken pods (OOMKilled, CrashLoopBackOff, ImagePullBackOff).
#
# Regenerate:  vhs docs/demo/autopsy.tape   (run from repo root, cluster up)

Output docs/demo/autopsy.gif

Set Shell zsh
Set FontSize 15
Set Width 1100
Set Height 760
Set Padding 18
Set Theme "Catppuccin Mocha"
Set TypingSpeed 45ms
Set PlaybackSpeed 1.0

# --- hidden setup: binary on PATH, clean prompt, no scrollback noise ---
Hide
Type "export PATH=$PWD:$PATH PS1='%F{green}$%f '" Enter
Type "clear" Enter
Show

# --- 1. Scan the namespace for trouble ---
Type "autopsy crashloops -n demo" Sleep 600ms Enter
Sleep 3s

# --- 2. Drill into the OOMKilled pod ---
Type "autopsy triage web-1 -n demo" Sleep 600ms Enter
Sleep 3.5s

# --- 3. Drill into the crash-looping worker (real logs + detected errors) ---
Type "autopsy triage worker-7 -n demo" Sleep 600ms Enter
Sleep 4s

Sleep 1.5s
