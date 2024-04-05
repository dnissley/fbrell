/**
 * Copyright (c) 2014-present, Facebook, Inc. All rights reserved.
 *
 * You are hereby granted a non-exclusive, worldwide, royalty-free license to use,
 * copy, modify, and distribute this software in source code or binary form for use
 * in connection with the web services and APIs provided by Facebook.
 *
 * As with any software that integrates with the Facebook platform, your use of
 * this software is subject to the Facebook Developer Principles and Policies
 * [http://developers.facebook.com/policy/]. This copyright notice shall be
 * included in all copies or substantial portions of the software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
 * FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
 * COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
 * IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
 * CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 */

// Package viewog implements HTTP handlers for /og and /rog* URLs on Rell.
package viewog

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	h "github.com/daaku/go.h"
	fb "github.com/daaku/go.h.js.fb"
	static "github.com/daaku/go.static"
	"github.com/fbsamples/fbrell/errcode"
	"github.com/fbsamples/fbrell/og"
	"github.com/fbsamples/fbrell/rellenv"
	"github.com/fbsamples/fbrell/view"
)

type Handler struct {
	Static       *static.Handler
	ObjectParser *og.Parser
}

// Handles /og/ requests.
func (a *Handler) Values(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	env, err := rellenv.FromContext(ctx)
	if err != nil {
		return err
	}
	values := r.URL.Query()
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) > 4 {
		return errcode.New(http.StatusNotFound, "Invalid URL: %s", r.URL.Path)
	}
	if len(parts) > 2 {
		values.Set("og:type", parts[2])
	}
	if len(parts) > 3 {
		values.Set("og:title", parts[3])
	}
	object, err := a.ObjectParser.FromValues(ctx, env, values)
	if err != nil {
		return err
	}
	_, err = h.Write(ctx, w, renderObject(ctx, env, a.Static, object))
	return err
}

// Handles /rog/* requests.
func (a *Handler) Base64(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	env, err := rellenv.FromContext(ctx)
	if err != nil {
		return err
	}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 3 {
		return errcode.New(http.StatusNotFound, "Invalid URL: %s", r.URL.Path)
	}
	object, err := a.ObjectParser.FromBase64(ctx, env, parts[2])
	if err != nil {
		return err
	}
	_, err = h.Write(ctx, w, renderObject(ctx, env, a.Static, object))
	return err
}

// Handles /rog-redirect/ requests.
func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 5 {
		return fmt.Errorf("Invalid URL: %s", r.URL.Path)
	}
	status, err := strconv.Atoi(parts[2])
	if err != nil || (status != 301 && status != 302) {
		return fmt.Errorf("Invalid status: %s", parts[2])
	}
	count, err := strconv.Atoi(parts[3])
	if err != nil {
		return fmt.Errorf("Invalid count: %s", parts[3])
	}
	context, err := rellenv.FromContext(ctx)
	if err != nil {
		return err
	}
	if count == 0 {
		http.Redirect(w, r, context.AbsoluteURL("/rog/"+parts[4]).String(), status)
	} else {
		count--
		url := context.AbsoluteURL(fmt.Sprintf(
			"/rog-redirect/%d/%d/%s", status, count, parts[4]))
		http.Redirect(w, r, url.String(), status)
	}
	return nil
}

// Renders <meta> tags for object.
func renderMeta(o *og.Object) h.HTML {
	var frag h.Frag
	for _, pair := range o.Pairs {
		frag = append(frag, &h.Meta{
			Property: pair.Key,
			Content:  pair.Value,
		})
	}
	return frag
}

// Auto linkify values that start with "http".
func renderValue(val string) h.HTML {
	txt := h.String(val)
	if strings.HasPrefix(val, "http") {
		return &h.A{HREF: val, Inner: txt}
	}
	return txt
}

// Renders a <table> with the meta data for the object.
func renderMetaTable(o *og.Object) h.HTML {
	var frag h.Frag
	for _, pair := range o.Pairs {
		frag = append(frag, &h.Tr{
			Inner: h.Frag{
				&h.Th{Inner: h.String(pair.Key)},
				&h.Td{Inner: renderValue(pair.Value)},
			},
		})
	}

	return &h.Table{
		Class: "table table-bordered table-striped og-info",
		Inner: h.Frag{
			&h.Thead{
				Inner: &h.Tr{
					Inner: h.Frag{
						&h.Th{Inner: h.String("Property")},
						&h.Th{Inner: h.String("Content")},
					},
				},
			},
			&h.Tbody{Inner: frag},
		},
	}
}

// Render a document for the Object.
func renderObject(ctx context.Context, env *rellenv.Env, s *static.Handler, o *og.Object) h.HTML {
	var title, header h.HTML
	if o.Title() != "" {
		title = &h.Title{h.String(o.Title())}
		header = &h.H1{
			Inner: &h.A{
				HREF:  o.URL(),
				Inner: h.String(o.Title()),
			},
		}
	}
	return &h.Document{
		Inner: h.Frag{
			&h.Head{
				Inner: h.Frag{
					&h.Meta{Charset: "utf-8"},
					title,
					&h.LinkStyle{
						HREF: "https://maxcdn.bootstrapcdn.com/twitter-bootstrap/2.2.0/css/bootstrap-combined.min.css",
					},
					&static.LinkStyle{
						HREF: view.DefaultPageConfig.Style,
					},
					renderMeta(o),
				},
			},
			&h.Body{
				Class: "container",
				Inner: h.Frag{
					&h.Div{ID: "fb-root"},
					view.DefaultPageConfig.GA,
					&fb.Init{
						URL:   env.SdkURL(),
						AppID: rellenv.FbApp(ctx).ID(),
					},
					&h.Div{
						Class: "row",
						Inner: h.Frag{
							&h.Div{
								Class: "span8",
								Inner: header,
							},
							&h.Div{
								Class: "span4",
								Inner: &h.A{
									Class: "btn btn-info pull-right",
									HREF:  o.LintURL(),
									Inner: h.Frag{
										&h.I{Class: "icon-warning-sign icon-white"},
										h.String(" Debugger"),
									},
								},
							},
						},
					},
					&h.Div{
						Class: "row",
						Inner: h.Frag{
							&h.Div{
								Class: "span6",
								Inner: h.Frag{
									renderMetaTable(o),
									&h.Iframe{
										Class: "like",
										Src:   o.LikeURL(),
									},
								},
							},
							&h.Div{
								Class: "span6",
								Inner: &h.A{
									HREF: o.ImageURL(),
									Inner: &h.Img{
										Src: o.ImageURL(),
										Alt: o.Title(),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
