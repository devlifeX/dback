<?php

if (!defined('ABSPATH')) {
    exit;
}

class DBack_Api_Key {

    const OPTION_NAME = 'dback_api_key';

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

    public static function is_valid($candidate) {
        if (!is_string($candidate) || '' === $candidate) {
            return false;
        }

        return hash_equals(self::get(), $candidate);
    }
}
