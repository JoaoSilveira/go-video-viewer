> [!WARNING]  
> Project made for personal use, use at your own risk

# Go Video Viewer

This is a server written in go for me to watch my anime. What are the requisites?

1. All video files are inside a folder
2. The modified time of the files is the order in which I'll watch
3. After watching I want to choose whether to keep, like or just mark as watched
4. If I choose not to keep a file, it'll be truncated (for me to remember not to download twice)
5. I want to be able to watch again files I keep
6. The database needs to be updated with the videos in the folder
   
# Progress

- [x] endpoints
  - [x] next in queue `/next-video`
  - [x] list saved `/video-list`
  - [x] watch saved `/watch/{id}`
  - [x] serve video file `/video/{id}`
  - [x] read folder to update database `/update`
- [x] respect parameters
- [x] truncate files
- [x] refactoring
- [x] make ini the same name as executable
- [x] automatically create ini file when it doesn't exist

## Motivation

I made this project first using svelte (version 4) + Bun, but the start time is eye boggling long (above 20s). So I used this opportunity to put my go skills into practice and try solving this minor inconvenience.

# Getting started

1. Download and install [go](https://go.dev/dl/) version 1.23.1 (or make it work on other version on your own)
2. Install [Templ](https://github.com/a-h/templ)
3. Generate templates
    ```
    templ generate
    ```
4. Build the project
    ```
    go build -o bin/
    ```
5. Create an `go-video-viewer.ini` file in the `/bin` folder (ini file must have the same name as the executable)
    ```ini
    database= # database path
    video_folder= # video folder path
    ```
6. Run the executable

## The ini file

The ini file can have the following properties:

| property | description |
| -------: | :---------- |
| database | The database file path. Ex: `database=videos.db`, `database=C:\Program Files\go-video-viewer\database.db` |
| video_folder | The path of the video folder. Ex: `video_folder=C:\Users\me\Videos`, `video_folder=videos` |
| address | (*OPTIONAL*) The address used to run the server (default on 127.0.0.1). Ex: `localhost`, `127.0.0.1` |
| port | (*OPTIONAL*) The port used to run the server (default on 3000). Ex: `8000`, `8080`, `16217` |

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