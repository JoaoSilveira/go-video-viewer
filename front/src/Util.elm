module Util exposing (..)

import Http
import Html exposing (a)
import Html exposing (text)
import Html exposing (Html)


type AsyncResource s e
    = Idle
    | Loading
    | Success s
    | Failure e
    | ReFetching (AsyncResource s e)


errorToString : Http.Error -> String
errorToString error =
    case error of
        Http.BadUrl string ->
            "Bad URL: " ++ string

        Http.Timeout ->
            "Timeout"

        Http.NetworkError ->
            "Network Error"

        Http.BadBody value ->
            "Bad body: " ++ value

        Http.BadStatus status ->
            "Bad Status: " ++ String.fromInt status


listSlice : Int -> Int -> List a -> List a
listSlice start len items =
    let
        recurse i remm l =
            case l of
                head :: tail ->
                    if i < start then
                        recurse (i + 1) remm tail

                    else if remm > 0 then
                        head :: recurse i (remm - 1) tail

                    else
                        []

                [] ->
                    []
    in
    recurse 0 len items

optionalList : List (Bool, a) -> List a
optionalList items =
    let
        recurse l =
            case l of
                (True, el) :: t ->
                    el :: recurse t

                (False, _) :: t ->
                    recurse t

                [] ->
                    []
    in
    recurse items
