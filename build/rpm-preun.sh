#!/bin/bash
# Remove polkit policy on uninstall
rm -f /usr/share/polkit-1/actions/com.legiz-ru.prizrak-box.policy 2>/dev/null || true
