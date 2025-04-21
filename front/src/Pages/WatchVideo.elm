module Pages.WatchVideo exposing (..)

import Browser
import Core.VideoViewer as VideoViewer
import Html
import Util exposing (..)
import VideoRepository exposing (..)
import Browser.Navigation as Navigation


type alias Model =
    { id : Int, viewerModel : VideoViewer.Model }


type Msg
    = VideoViewerMsg VideoViewer.Msg


init : Navigation.Key -> Int -> ( Model, Cmd Msg )
init key id =
    let
        ( model, cmd ) =
            VideoViewer.init key True (getVideoById id)
    in
    ( { id = id, viewerModel = model }
    , Cmd.map VideoViewerMsg cmd
    )


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        VideoViewerMsg videoMsg ->
            let
                ( newModel, cmd ) =
                    VideoViewer.update videoMsg model.viewerModel
            in
            ( {model | viewerModel = newModel }, Cmd.map VideoViewerMsg cmd )


view : Model -> Browser.Document Msg
view model =
    { title = "Watching video " ++ String.fromInt model.id
    , body = [ Html.map VideoViewerMsg <| VideoViewer.view model.viewerModel ]
    }
