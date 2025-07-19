module Routes exposing (..)

import Url exposing (Url)
import Url.Parser exposing ((</>), (<?>), Parser, int, map, oneOf, parse, s, top)
import Url.Parser.Query as Query


type Route
    = Home
    | NextVideo
    | WatchVideo Int
    | SavedList (Maybe Int)
      --  | Backlog
    | ErrorPage


routeParser : Parser (Route -> a) a
routeParser =
    oneOf
        [ map Home top
        , map NextVideo (s "next-video")
        , map WatchVideo (s "watch" </> int)
        , map SavedList (s "saved-list" <?> Query.int "page")
        ]


routeFromUrl : Url -> Route
routeFromUrl url =
    parse routeParser url
        |> Maybe.withDefault ErrorPage


routeHref : Route -> String
routeHref route =
    case route of
        Home ->
            "/"

        NextVideo ->
            "/next-video"

        ErrorPage ->
            "/"

        WatchVideo id ->
            "/watch/" ++ String.fromInt id

        SavedList _ ->
            "/saved-list"


isRouteActive : Url -> Route -> Bool
isRouteActive url route =
    let
        urlRoute =
            routeFromUrl url
    in
    urlRoute == route


areUrlsSamePage : Url -> Url -> Bool
areUrlsSamePage a b =
    case (routeFromUrl a, routeFromUrl b) of
        (Home, Home) ->
            True

        (NextVideo, NextVideo) ->
            True

        (WatchVideo _, WatchVideo _) ->
            True

        (SavedList _, SavedList _) ->
            True

        (ErrorPage, ErrorPage) ->
            True

        (_, _) ->
            False
