# Go Video Viewer

This is a server written in go for me to watch my anime. What are the requisites?

1. All video files are inside a folder
2. The modified time of the files is the order in which I'll watch
3. After watching I want to choose whether to keep, like or just mark as watched
4. If I choose not to keep a file, it'll be truncated (for me to remember not to download twice)
5. I want to be able to watch again files I keep
   
# Progress

- [x] endpoints
  - [x] next in queue `/next-video`
  - [x] list saved `/video-list`
  - [x] watch saved `/watch/{id}`
- [ ] respect parameters
- [ ] truncate files
- [ ] serve video folder
- [ ] refactoring

## Motivation

I made this project first using svelte (version 4) + Bun, but the start time is eye boggling long (above 20s). So I used this opportunity to put my go skills into practice and try improving this thing.

# How do I use this?

1. Get all dependencies
    ```
    go get
    ```
2. Generate templates
    ```
    templ generate
    ```
3. Build the project
    ```
    go build -o /bin/go-video-viewer.exe
    ```
4. Create an `ini` file in the `/bin` folder
    ```ini
    database= # database path
    video_folder= # video folder path
    ```
5. Run the executable

## Migrating from the previous project

In the previous version of this project, the "database" was a JSON file with the following schema:

```json
{
    "watched": [
        {
            "name": "file name + extension",
            "date": "some ISO format date",
            "favorited": true, // omitted if false
            "save": true // ommitted if false
        },
        ...
    ],
    "toWatch": [
        {
            "name": "file name + extension",
            "date": "some ISO format date",
            "favorited": true, // omitted if false
            "save": true // ommitted if false
        },
        ...
    ],
    "current": {
        "name": "file name + extension",
        "date": "some ISO format date",
        "favorited": true, // omitted if false
        "save": true // ommitted if false
    }
}
```

So to import this data into this project you can inform the command line argument `json-file`, using it like this:

```bash
.\go-video-viewer.exe --json-file "path to the json file"
```