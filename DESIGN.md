---
name: Quick Feed
description: A personal reading space with a calm editorial feel and a bright blue accent.
colors:
  primary: "#2563eb"
  on-primary: "#ffffff"
  brand: "#1d4ed8"
  background: "oklch(1 0 0)"
  foreground: "oklch(0.145 0 0)"
  secondary: "#dbeafe"
  on-secondary: "oklch(0.205 0 0)"
  muted: "oklch(0.97 0 0)"
  muted-foreground: "oklch(0.556 0 0)"
  border: "oklch(0.922 0 0)"
  destructive: "oklch(0.577 0.245 27.325)"
typography:
  heading:
    fontFamily: "IBM Plex Sans Thai, sans-serif"
    fontSize: 2.25rem
    fontWeight: 700
    lineHeight: 1.1
    letterSpacing: "-0.025em"
  body:
    fontFamily: "IBM Plex Sans Thai, sans-serif"
    fontSize: 0.875rem
    fontWeight: 400
    lineHeight: 1.5
  metadata:
    fontFamily: "IBM Plex Sans Thai, sans-serif"
    fontSize: 0.75rem
    fontWeight: 400
    lineHeight: 1.5
  label:
    fontFamily: "IBM Plex Sans Thai, sans-serif"
    fontSize: 0.625rem
    fontWeight: 700
    letterSpacing: 0.1em
rounded:
  sm: 0.375rem
  md: 0.5rem
  lg: 0.625rem
  xl: 0.875rem
spacing:
  sm: 0.5rem
  md: 1rem
  lg: 1.5rem
  xl: 2rem
components:
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.on-primary}"
    rounded: "{rounded.md}"
  button-secondary:
    backgroundColor: "{colors.secondary}"
    textColor: "{colors.on-secondary}"
    rounded: "{rounded.md}"
  badge-secondary:
    backgroundColor: "{colors.secondary}"
    textColor: "{colors.on-secondary}"
  reading-surface:
    backgroundColor: "{colors.background}"
    textColor: "{colors.foreground}"
---

## Overview

Quick Feed is a personal reading space: calm, friendly, and lightly editorial. The content leads. Generous white space, compact metadata, fine dividers, and small playful details make a collection of articles feel welcoming without competing with the reading list.

This document describes the current light appearance of the feed mockup. The stylesheet contains dark-theme values, but a complete dark experience has not yet been established. The implementation in `frontend/src/styles.css` and the existing components remain the source of truth for exact values.

## Colors

- **Primary blue** (`#2563eb`) gives the brand its energy. Use it for prominent interactive surfaces and small visual accents, paired with white text.
- **Brand blue** (`#1d4ed8`) highlights short headings and decorative details on light backgrounds.
- **Pale blue** (`#dbeafe`) marks selected navigation, secondary badges, and helpful callouts without overpowering content.
- **White and near-black** provide the main reading surface and strong text hierarchy. Muted gray supports descriptions, timestamps, and other secondary information; a light border separates regions and feed items.
- Source avatars and article artwork may use restrained supporting colors to distinguish content. These colors are content accents, not replacements for the core palette.

Check text and control contrast in context, particularly for small muted text and blue on white.

## Typography

Use IBM Plex Sans Thai throughout, with a sans-serif fallback. It supports the interface's mix of Thai and English. Headings are bold and compact; article titles are prominent but smaller than the page title. Body and description text should remain comfortable to read, with relaxed line height for longer descriptions.

Use small, bold, widely tracked uppercase English labels sparingly as editorial accents. Keep metadata quieter than titles, and do not rely on small decorative labels to communicate essential information.

## Layout

The feed is the central focus. On mobile, use a compact header, a single content column, and bottom navigation. At the `md` breakpoint, introduce a left navigation sidebar; at `xl`, add a right-hand supporting sidebar. The feed content is centered with a maximum width of 910px and responsive horizontal padding.

Give sections room to breathe. Present articles as a vertical list separated by subtle horizontal rules, with source and time above the title, a short description, and actions and metadata below. Small artwork sits beside the text when space allows. Preserve readable text and usable controls as the viewport narrows.

## Elevation & Depth

Prefer flat surfaces, light borders, and background contrast to heavy shadows. A pale blue callout can group helpful information; a near-black feature card provides an occasional strong contrast in supporting space. Keep the article list visually open.

## Shapes

Use softly rounded corners for buttons, cards, and artwork. The base radius is `0.625rem`, with related sizes derived from it in the stylesheet. Circular shapes suit source avatars, the brand mark, and compact bookmark controls. Badges are pill-shaped.

## Components

- **Buttons:** Use the existing button variants for primary actions, pale-blue selected states, quiet ghost actions, and outlined icon actions. Preserve visible keyboard focus, disabled states, and accessible labels for icon-only controls.
- **Badges:** Use compact pale-blue pills for categories and counts. Keep labels brief.
- **Cards:** Use cards selectively for grouped or supporting content, such as the welcome callout and dark promotional panel, rather than enclosing every article.
- **Article rows:** Prioritize source, title, and description. Keep read and saved states distinguishable through text or icon treatment as well as color.
- **Source avatars and artwork:** Use simple colored circles with initials for sources and small, rounded, colored blocks for article artwork. Supporting colors can vary by content.
- **Navigation:** Show the selected destination clearly. Adapt the navigation to the available width while keeping the feed and saved items easy to reach.

## Do's and Don'ts

- Do give article titles and content the strongest visual hierarchy.
- Do use blue deliberately for brand, selection, and emphasis, with neutral space around it.
- Do retain clear focus indicators, meaningful labels, and legible contrast.
- Don't add heavy shadows or a separate card around every feed item.
- Don't let decorative uppercase labels or artwork compete with the reading content.
- Don't treat the mockup's sample sources, counts, or artwork as fixed product data.
