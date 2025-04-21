module Pages.NextVideo exposing (..)

import Browser
import Core.VideoViewer as VideoViewer
import Html
import Util exposing (..)
import VideoRepository exposing (..)
import Browser.Navigation as Navigation


type alias Model = VideoViewer.Model
type Msg =
    VideoViewerMsg VideoViewer.Msg


init : Navigation.Key -> ( Model, Cmd Msg )
init key =
    let
        (model, cmd) = VideoViewer.init key False getNextVideo
    in
    ( model, Cmd.map VideoViewerMsg cmd)


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        VideoViewerMsg videoMsg ->
            let
                (newModel, cmd) = VideoViewer.update videoMsg model
            in
            (newModel, Cmd.map VideoViewerMsg cmd)


view : Model -> Browser.Document Msg
view model =
    { title = "Video Queue"
    , body = [ Html.map VideoViewerMsg <| VideoViewer.view model ]
    }
