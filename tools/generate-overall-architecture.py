#!/usr/bin/env python3
# Copyright 2023 Red Hat, Inc
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# vim: set fileencoding=utf-8

"""Simple preprocessor for generating area maps for Overall Architecture page."""

import PIL.Image as Image
from PIL import ImageDraw
from pathlib import Path

template_file = "overall-architecture-template.html"
output_file = "overall-architecture.html"
areas_file = "areas.txt"
output_directory = ""

input_image = "Overall_architecture_in.png"
output_image = "Overall_architecture.png"


def load_text_file(filename):
    with open(filename, "r") as fin:
        return fin.read()


def load_file_as_lines(filename):
    with open(areas_file, "r") as fin:
        return fin.read().splitlines()


def save_text_file(filename, content):
    with open(filename, "w") as fout:
        fout.write(content)


def make_path(node_type, node, suffix):
    return node_type + "/" + node.lower().replace(" ", "-") + suffix


def make_path_to_markdown_file(node_type, node):
    return make_path(node_type, node, ".md")


def make_href(node_type, node):
    return make_path(node_type, node, ".html")


def format_area(x, y, width, height, node, href):
    space = 16*" "
    return f'{space}<area shape="rect" coords="{x}, {y}, {x+width}, {y+height}" title="{node}" alt="{node}" href="{href}" />\n'


def generate_area_maps(areas):
    area_maps = ""
    for area in areas:
        splitted = area.split(" ")
        node_type = splitted[0]
        x = int(splitted[1])
        y = int(splitted[2])
        width = int(splitted[3])
        height = int(splitted[4])
        node = " ".join(splitted[5:])
        href = make_href(node_type, node)
        area_maps += format_area(x, y, width, height, node, href)
    return area_maps


def draw_areas(input_image_file_name, output_image_file_name, areas):
    colors = {
            "component":"#80008020",
            "channel":"#00800020",
            "topic":"#00008020",
            "storage":"#80000020",
            "interface":"#80800020",
            }

    # we need to open the image and get rid of the orinal alpha channel
    # because draw.io put meaningles information there
    image = Image.open(input_image_file_name).convert("RGB")
    draw = ImageDraw.Draw(image, "RGBA")

    for area in areas:
        splitted = area.split(" ")
        node_type = splitted[0]
        x = int(splitted[1])
        y = int(splitted[2])
        width = int(splitted[3])
        height = int(splitted[4])
        color = colors[node_type]
        draw.rectangle((x, y, x+width, y+height), outline="black", fill=color)

    image.save(output_image_file_name)


def touch_files(directory, areas):
    for area in areas:
        splitted = area.split(" ")
        node_type = splitted[0]
        node = " ".join(splitted[5:])
        path = Path(directory, make_path_to_markdown_file(node_type, node))
        path.touch()


def main():
    template = load_text_file(template_file)
    areas = load_file_as_lines(areas_file)
    area_maps = generate_area_maps(areas)
    html_page = template.replace("<map-areas />", area_maps[:-1])
    save_text_file(output_file, html_page)
    draw_areas(input_image, output_image, areas)
    touch_files(output_directory, areas)


if __name__ == "__main__":
    main()
