# Setup

- Add a line below to a file ./.bash_profile

```
export GOPRIVATE="gitlab.com/connect.club"
```

- Add to ~/.gitconfig next fragment

```
[url "ssh://git@gitlab.com/"]
    insteadOf = https://gitlab.com/
```

# Update dependency

### jvbuster-client

- go get gitlab.com/connect.club/jitsi/connectclub-jvbuster-client.git

# Build

- Update paths in build.sh
- Run `build.sh`