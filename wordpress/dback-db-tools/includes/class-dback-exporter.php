<?php

use Ifsnop\Mysqldump\Mysqldump;

if (!defined('ABSPATH')) {
    exit;
}

class DBack_Exporter {

    /**
     * Stream a gzip-compressed SQL dump directly to the client.
     *
     * @throws Exception
     */
    public static function stream_gzip() {
        if (!function_exists('deflate_init')) {
            throw new RuntimeException('The zlib extension is required for gzip export.');
        }

        if (DBack_Database::has_pdo_mysql()) {
            self::stream_gzip_with_mysqldump();
        }

        DBack_Exporter_Mysqli::stream_gzip();
    }

    /**
     * @throws Exception
     */
    private static function stream_gzip_with_mysqldump() {
        @set_time_limit(0);
        if (function_exists('ini_set')) {
            @ini_set('memory_limit', '512M');
        }

        while (ob_get_level() > 0) {
            ob_end_clean();
        }

        if (function_exists('apache_setenv')) {
            @apache_setenv('no-gzip', '1');
        }

        @ini_set('zlib.output_compression', 'Off');

        $filename = 'dback-export-' . gmdate('Y-m-d-His') . '.sql.gz';

        header('Content-Type: application/gzip');
        header('Content-Disposition: attachment; filename="' . $filename . '"');
        header('Cache-Control: no-store, no-cache, must-revalidate');
        header('Pragma: no-cache');
        header('X-Content-Type-Options: nosniff');

        $creds = DBack_Database::credentials();
        $dump_settings = array(
            'compress' => Mysqldump::GZIPSTREAM,
            'default-character-set' => Mysqldump::UTF8MB4,
            'add-drop-table' => true,
            'single-transaction' => true,
            'lock-tables' => false,
            'hex-blob' => true,
            'extended-insert' => true,
            'disable-keys' => true,
            'add-locks' => true,
            'skip-comments' => false,
            'skip-dump-date' => false,
        );

        $pdo_settings = DBack_Database::pdo_options(false);

        $dump = new Mysqldump(
            DBack_Database::dsn(),
            $creds['user'],
            $creds['pass'],
            $dump_settings,
            $pdo_settings
        );

        $dump->start('php://output');
        exit;
    }
}
