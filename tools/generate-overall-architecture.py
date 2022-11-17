#!/usr/bin/env python3
# vim: set fileencoding=utf-8

"""Simple preprocessor for generating area maps for Overall Architecture page."""

template_file = "overall-architecture-template.html"
output_file = "overall-architecture.html"
areas_file = "areas.txt"


def load_text_file(filename):
    with open(filename, "r") as fin:
        return fin.read()


def load_file_as_lines(filename):
    with open(areas_file, "r") as fin:
        return fin.read().splitlines()


def save_text_file(filename, content):
    with open(filename, "w") as fout:
        fout.write(content)


def make_href(node_type, node):
    return node_type + "/" + node.lower().replace(" ", "-") + ".html"


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


def main():
    template = load_text_file(template_file)
    areas = load_file_as_lines(areas_file)
    area_maps = generate_area_maps(areas)
    html_page = template.replace("<map-areas />", area_maps[:-1])
    save_text_file(output_file, html_page)


if __name__ == "__main__":
    main()
