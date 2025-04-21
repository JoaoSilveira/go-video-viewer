module VideoRepository exposing (..)

import Http
import Iso8601
import Json.Decode as Decode
import Json.Encode as Encode
import Time


type VideoStatus
    = Unwatched
    | Watched
    | Liked
    | Saved


type alias Video =
    { id : Int
    , filename : String
    , nickname : Maybe String
    , tags : List String
    , created_at : Time.Posix
    , status : VideoStatus
    }


type alias VideoUpdatePayload =
    { nickname : String
    , tags : List String
    , status : VideoStatus
    }


type alias VideoInfo =
    { video : Video
    , next : Maybe Video
    }


getNextVideo : (Result Http.Error VideoInfo -> msg) -> Cmd msg
getNextVideo msg =
    Http.get
        { url = "/api/video/next"
        , expect = Http.expectJson msg videoInfoDecoder
        }


getVideoById : Int -> (Result Http.Error VideoInfo -> msg) -> Cmd msg
getVideoById id msg =
    Http.get
        { url = "/api/video/" ++ String.fromInt id
        , expect = Http.expectJson msg videoInfoDecoder
        }


updateVideo : Int -> VideoUpdatePayload -> (Result Http.Error () -> msg) -> Cmd msg
updateVideo id video msg =
    let
        jsonPayload =
            Encode.object
                [ ( "nickname", Encode.string video.nickname )
                , ( "status", Encode.int <| encodeStatus video.status )
                , ( "tags", Encode.list Encode.string video.tags )
                ]
    in
    Http.post
        { url = "/api/video/" ++ String.fromInt id
        , expect = Http.expectWhatever msg
        , body = Http.jsonBody jsonPayload
        }


listSavedVideos : (Result Http.Error (List Video) -> msg) -> Cmd msg
listSavedVideos msg =
    Http.get
        { url = "/api/video/list"
        , expect = Http.expectJson msg videoListDecoder
        }


encodeStatus : VideoStatus -> Int
encodeStatus status =
    case status of
        Unwatched ->
            1

        Watched ->
            2

        Liked ->
            3

        Saved ->
            4


videoStatusDecoder : Decode.Decoder VideoStatus
videoStatusDecoder =
    Decode.int
        |> Decode.andThen
            (\statusCode ->
                case statusCode of
                    1 ->
                        Decode.succeed Unwatched

                    2 ->
                        Decode.succeed Watched

                    3 ->
                        Decode.succeed Liked

                    4 ->
                        Decode.succeed Saved

                    _ ->
                        Decode.fail ("Invalid VideoStatus: " ++ String.fromInt statusCode)
            )


videoDecoder : Decode.Decoder Video
videoDecoder =
    Decode.map6 Video
        (Decode.field "id" Decode.int)
        (Decode.field "filename" Decode.string)
        (Decode.field "nickname" (Decode.nullable Decode.string))
        (Decode.field "tags" (Decode.list Decode.string))
        (Decode.field "created_at" Iso8601.decoder)
        (Decode.field "status" videoStatusDecoder)


videoListDecoder : Decode.Decoder (List Video)
videoListDecoder =
    Decode.field "videos" <|
        Decode.list videoDecoder


videoInfoDecoder : Decode.Decoder VideoInfo
videoInfoDecoder =
    Decode.map2 VideoInfo
        (Decode.field "video" videoDecoder)
        (Decode.field "next" (Decode.nullable videoDecoder))
