<?php

if (!defined('ABSPATH')) {
    exit;
}

class DBack_Importer {

    /**
     * Import SQL from a gzip-compressed file path.
     *
     * @param string $path
     * @return array{success:bool,statements_executed:int}
     * @throws Exception
     */
    public static function import_gzip_file($path) {
        if (!is_readable($path)) {
            throw new RuntimeException('Import file is not readable.');
        }

        if (!function_exists('gzopen')) {
            throw new RuntimeException('The zlib extension is required for gzip import.');
        }

        @set_time_limit(0);
        if (function_exists('ini_set')) {
            @ini_set('memory_limit', '512M');
        }

        $handle = gzopen($path, 'rb');
        if (false === $handle) {
            throw new RuntimeException('Unable to open gzip import file.');
        }

        try {
            $executed = self::execute_stream($handle);
        } finally {
            gzclose($handle);
        }

        return array(
            'success' => true,
            'statements_executed' => $executed,
        );
    }

    /**
     * Import SQL from the raw request body (gzip stream).
     *
     * @return array{success:bool,statements_executed:int}
     * @throws Exception
     */
    public static function import_request_body() {
        @set_time_limit(0);
        if (function_exists('ini_set')) {
            @ini_set('memory_limit', '512M');
        }

        $temp_file = DBack_Database::temp_file('import', '.sql.gz');
        $input = fopen('php://input', 'rb');
        $output = fopen($temp_file, 'wb');

        if (false === $input || false === $output) {
            throw new RuntimeException('Unable to read import payload.');
        }

        try {
            stream_copy_to_stream($input, $output);
        } finally {
            fclose($input);
            fclose($output);
        }

        try {
            return self::import_gzip_file($temp_file);
        } finally {
            if (file_exists($temp_file)) {
                unlink($temp_file);
            }
        }
    }

    /**
     * @param resource $handle
     * @return int
     */
    private static function execute_stream($handle) {
        $buffer = '';
        $executed = 0;

        while (!feof($handle)) {
            $line = fgets($handle);
            if (false === $line) {
                break;
            }

            $trimmed = trim($line);
            if ('' === $trimmed || 0 === strpos($trimmed, '--') || 0 === strpos($trimmed, '#')) {
                continue;
            }

            $buffer .= $line;

            if (';' === substr(rtrim($line), -1)) {
                DBack_Database::exec($buffer);
                $buffer = '';
                $executed++;
            }
        }

        if ('' !== trim($buffer)) {
            DBack_Database::exec($buffer);
            $executed++;
        }

        return $executed;
    }
}
