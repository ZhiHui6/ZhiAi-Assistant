# Changelog

## [1.2.0]

### Added

- New "Ring animation" setting: while an alarm or countdown rings, the panel edge glows red. Choose Breathing pulse, Sweeping light, Double flash or Off, and preview it right from the settings page.
- New "Max alarms" setting (1–8, default 8); the Add button in the sidebar greys out once the limit is reached.
- New Show Qoder / WorkBuddy / Trae switches in Settings let you hide individual platforms from the Usage tab.
- "Notes" is now "Memos": the UI, code identifiers and data file were renamed together. An existing notes.json migrates to memos.json on first launch, and older backups still import.
- Memos now accept attachments: images, audio, video, text files and links. Use the paperclip menu to pick files or add a link, or drop files straight onto a memo card. Attachments are copied into the data folder and included in backups.
- New Memos section in Settings with a per-attachment size limit (100 MB / 200 MB / 500 MB / 1 GB / 2 GB / Unlimited).
- Memo attachments preview on hover: a thumbnail panel appears above the filename tip — the picture for images, the opening lines for text and the 30th frame for videos.
- Attachments can also be added by pasting clipboard images or files into the memo input box, or by dropping files onto a memo card.

### Changed

- Error messages, tray menus and file dialogs are now available in English and Traditional Chinese, switching live with the interface language.

## [1.1.0]

### Added

- Alarm time now uses a visual time picker; hours and minutes can be dragged or stepped.
- Countdown hours, minutes and seconds get numeric stepper buttons that also respond to the keyboard and mouse wheel.
- Alarms and countdowns now use collapsible setup panels with Start and Cancel buttons; a collapsed panel shows only the current status.
- The window can dock to the desktop layer and restores itself automatically when the foreground window changes.
- Image files in the shelf now show thumbnails, so you can tell them apart without opening them.
- Notes display their tags next to the timestamp in the static view; hovering fades the tags out and reveals color, pin and delete actions instead.
- Shelf group headers show the number of files in the group, e.g. `[2]`.

### Changed

- Unified the timer area styling; setup panels and lists use tighter spacing and colors.
- All tooltips now use the same custom styled bubble, with long file paths wrapped; works in the settings window too.
- Tightened vertical spacing in the shelf and mounts; dividers now keep a uniform 4px gap from the content above and below.
- The shelf view-mode button now uses a grid icon for clearer meaning.
- Reduce motion now also covers the tag crossfade and tooltip animations.

### Fixed

- Fixed the "Desktop Files" breadcrumb remaining in the header after returning from file browsing.

## [1.0.0]

First public release.

[1.2.0]: https://github.com/ZhiHui6/ZhiAi-Assistant/releases/tag/v1.2.0
[1.1.0]: https://github.com/ZhiHui6/ZhiAi-Assistant/releases/tag/v1.1.0
[1.0.0]: https://github.com/ZhiHui6/ZhiAi-Assistant/releases/tag/v1.0.0
