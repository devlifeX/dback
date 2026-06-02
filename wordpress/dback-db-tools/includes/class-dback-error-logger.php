<?php

if (!defined('ABSPATH')) {
    exit;
}

class DBack_Error_Logger {

    const OPTION_KEY = 'dback_error_log';
    const MAX_ENTRIES = 100;
    const LOG_FILENAME = 'dback-errors.log';

    /**
     * @param string $level
     * @param string $code
     * @param string $message
     * @param array<string,mixed> $context
     * @return string
     */
    public static function log($level, $code, $message, $context = array()) {
        $entry = array(
            'id' => wp_generate_password(8, false, false),
            'time' => gmdate('c'),
            'level' => $level,
            'code' => $code,
            'message' => $message,
            'context' => self::sanitize_context($context),
        );

        self::append_file_entry($entry);
        self::store_entry($entry);

        return $entry['id'];
    }

    /**
     * @param string $code
     * @param Throwable $exception
     * @param array<string,mixed> $context
     * @return string
     */
    public static function log_exception($code, $exception, $context = array()) {
        return self::log(
            'error',
            $code,
            $exception->getMessage(),
            array_merge(
                $context,
                array(
                    'exception' => get_class($exception),
                    'file' => $exception->getFile(),
                    'line' => $exception->getLine(),
                )
            )
        );
    }

    /**
     * @param string $operation
     * @param string $code
     * @param Throwable $exception
     * @return WP_Error
     */
    public static function to_wp_error($operation, $code, $exception) {
        $error_id = self::log_exception(
            $code,
            $exception,
            array(
                'operation' => $operation,
                'source' => self::detect_source(),
            )
        );

        return new WP_Error(
            $code,
            $exception->getMessage(),
            array(
                'status' => self::http_status_for($exception),
                'error_id' => $error_id,
                'operation' => $operation,
                'logged_at' => gmdate('c'),
                'details' => array(
                    'exception' => get_class($exception),
                ),
            )
        );
    }

    /**
     * @param int $limit
     * @return array<int,array<string,mixed>>
     */
    public static function get_entries($limit = 50) {
        $entries = get_option(self::OPTION_KEY, array());
        if (!is_array($entries)) {
            return array();
        }

        return array_slice($entries, 0, max(1, (int) $limit));
    }

    public static function clear() {
        delete_option(self::OPTION_KEY);

        $path = self::log_file_path();
        if (is_file($path)) {
            unlink($path);
        }
    }

    public static function log_file_path() {
        try {
            $dir = DBack_Database::temp_dir();
        } catch (Throwable $exception) {
            $dir = sys_get_temp_dir();
        }

        return trailingslashit($dir) . self::LOG_FILENAME;
    }

    /**
     * @param array<string,mixed> $entry
     */
    private static function append_file_entry($entry) {
        $path = self::log_file_path();
        $line = wp_json_encode($entry);
        if (false === $line) {
            return;
        }

        file_put_contents($path, $line . PHP_EOL, FILE_APPEND | LOCK_EX);
    }

    /**
     * @param array<string,mixed> $entry
     */
    private static function store_entry($entry) {
        $entries = get_option(self::OPTION_KEY, array());
        if (!is_array($entries)) {
            $entries = array();
        }

        array_unshift($entries, $entry);
        $entries = array_slice($entries, 0, self::MAX_ENTRIES);
        update_option(self::OPTION_KEY, $entries, false);
    }

    /**
     * @param array<string,mixed> $context
     * @return array<string,mixed>
     */
    private static function sanitize_context($context) {
        $safe = array();

        foreach ($context as $key => $value) {
            if (is_scalar($value) || null === $value) {
                $safe[$key] = $value;
                continue;
            }

            if (is_array($value)) {
                $safe[$key] = self::sanitize_context($value);
            }
        }

        return $safe;
    }

    private static function detect_source() {
        if (defined('REST_REQUEST') && REST_REQUEST) {
            return 'rest_api';
        }

        if (is_admin()) {
            return 'admin';
        }

        return 'unknown';
    }

    /**
     * @param Throwable $exception
     * @return int
     */
    private static function http_status_for($exception) {
        if ($exception instanceof InvalidArgumentException) {
            return 400;
        }

        return 500;
    }
}
