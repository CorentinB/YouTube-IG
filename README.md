![forthebadge](https://forthebadge.com/images/badges/built-with-love.svg)
![forthebadge](https://forthebadge.com/images/badges/made-with-go.svg)

# YouTube-IG
💾 Light and fast YouTube video IDs grabber.

# Usage

First download the latest release from https://github.com/CorentinB/youtube-ma/releases
Extract it and make it executable with:
```
chmod +x YouTube-IG
```

YT-IG takes a list as a parameter, with at least 1 ID in it, and a concurrency parameter.
```
./YouTube-IG ids.txt 32
```

Here **32** is the number of goroutines maximum that can be run at the same time, it'll depend on your system, as it's also linked to a certain number of files opened at the same time, that could be limited by your system's configuration. If you want to use a bigger value, tweak your system, such as **ulimit**.
Default for this value if you don't precise any value is **16**, should be safe in most system.
