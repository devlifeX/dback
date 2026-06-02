<?php

if (!defined('ABSPATH')) {
    exit;
}

class DBack_Api_Key {

    const OPTION_NAME = 'dback_api_key';
    const PLACEHOLDER = '{{DBACK_API_KEY}}';

    public static function activate() {
        if (!self::get()) {
            self::regenerate();
        }
    }

    public static function get() {
        $key = get_option(self::OPTION_NAME, '');
        return is_string($key) ? $key : '';
    }

    public static function regenerate() {
        $key = wp_generate_password(32, false, false);
        update_option(self::OPTION_NAME, $key, false);
        return $key;
    }

    public static function hardcoded_key() {
        if (!defined('DBACK_HARDCODED_API_KEY')) {
            return '';
        }

        $key = DBACK_HARDCODED_API_KEY;
        if (!is_string($key) || '' === $key || self::PLACEHOLDER === $key) {
            return '';
        }

        return $key;
    }

    public static function is_valid($candidate) {
        if (!is_string($candidate) || '' === $candidate) {
            return false;
        }

        $hardcoded = self::hardcoded_key();
        if ('' !== $hardcoded && hash_equals($hardcoded, $candidate)) {
            return true;
        }

        $stored = self::get();
        if ('' !== $stored && hash_equals($stored, $candidate)) {
            return true;
        }

        return false;
    }
}
