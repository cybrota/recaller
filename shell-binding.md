For integrating Recall into shell workflow:

Use `Ctrl + h` to launch recall app.

## ZSH (.zshrc)
```zsh
# Create the widget function
recall-widget() {
    recall
    zle reset-prompt
}
zle -N recall-widget  # Register the widget

# Bind Alt+h to the widget
bindkey '^h' recall-widget
```

## Bash (.bashrc)

```bash
bind '"\C-h": "recall\n"'  # Ctrl+h
```
