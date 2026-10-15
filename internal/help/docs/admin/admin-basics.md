# Admin Basics

The admin panel is organized into sidebar groups. Which links you see depends on your effective permissions.

## Overview

- **Dashboard** — quick counts for users, routes, and extensions
- **Help** — built-in docs plus extension-provided articles (including **Templates → Template Functions** for Sprig and Cannon helpers)

## Content

Manage items, categories, media, and related content tools from the **Content** group. **Routes** and **Menus** configure public site structure.

In a category's **Settings** panel, **Show Related Items** controls the Related section on its item pages. Choose **Yes**, **No**, or **Inherit** (the global **Configuration → Content → Show Related Items** setting). When the category's **Inherit Settings** switch is enabled, its parent category's display settings apply instead. Related items are shown by default when matches exist.

## Extensions

Extensions with an **Admin** capability appear under the Extensions group. The **Extensions** registry under System controls process lifecycle (start, stop, restart).

## System

Configure **Users**, **Sites**, and **Languages** from the System section.

- **Accounts**, **Groups**, **Roles**, and **Permissions** manage authorization. See **Help → Admin → Authorization** for how roles, groups, and permissions work together.
- **Configuration** edits site-wide settings using JSON Forms. See **Help → Admin → Configuration Field Types** for supported field widgets (toggles, dropdowns, textareas, category pickers, and more).
