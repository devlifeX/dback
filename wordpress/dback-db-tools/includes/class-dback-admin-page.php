<?php

if (!defined('ABSPATH')) {
    exit;
}

class DBack_Admin_Page {

    public function register_menu() {
        add_management_page(
            __('DBack DB Tools', 'dback-db-tools'),
            __('DBack DB Tools', 'dback-db-tools'),
            'manage_options',
            'dback-db-tools',
            array($this, 'render_page')
        );
    }

    public function enqueue_assets($hook) {
        if ('tools_page_dback-db-tools' !== $hook) {
            return;
        }

        wp_enqueue_script(
            'dback-db-tools-admin',
            DBACK_DB_TOOLS_URL . 'assets/admin.js',
            array(),
            DBACK_DB_TOOLS_VERSION,
            true
        );

        wp_localize_script(
            'dback-db-tools-admin',
            'dbackDbTools',
            array(
                'restRoot' => esc_url_raw(rest_url(DBACK_DB_TOOLS_REST_NAMESPACE)),
                'nonce' => wp_create_nonce('wp_rest'),
                'apiKey' => DBack_Api_Key::get(),
                'database' => DB_NAME,
                'strings' => array(
                    'exportStarted' => __('Starting export...', 'dback-db-tools'),
                    'exportDone' => __('Export completed.', 'dback-db-tools'),
                    'importStarted' => __('Importing database...', 'dback-db-tools'),
                    'importDone' => __('Import completed.', 'dback-db-tools'),
                    'queryRunning' => __('Running query...', 'dback-db-tools'),
                    'queryDone' => __('Query completed.', 'dback-db-tools'),
                    'genericError' => __('Something went wrong.', 'dback-db-tools'),
                    'fileRequired' => __('Please choose a .sql.gz file.', 'dback-db-tools'),
                    'sqlRequired' => __('Please enter a SQL query.', 'dback-db-tools'),
                    'logsLoading' => __('Loading error log...', 'dback-db-tools'),
                    'logsLoaded' => __('Error log refreshed.', 'dback-db-tools'),
                    'logsCleared' => __('Error log cleared.', 'dback-db-tools'),
                    'logsEmpty' => __('No errors logged yet.', 'dback-db-tools'),
                ),
            )
        );
    }

    public function render_page() {
        if (!current_user_can('manage_options')) {
            wp_die(esc_html__('You do not have permission to access this page.', 'dback-db-tools'));
        }

        if (
            isset($_POST['dback_regenerate_key']) &&
            check_admin_referer('dback_regenerate_key')
        ) {
            DBack_Api_Key::regenerate();
            echo '<div class="notice notice-success is-dismissible"><p>' .
                esc_html__('API key regenerated.', 'dback-db-tools') .
                '</p></div>';
        }

        $api_key = DBack_Api_Key::get();
        ?>
        <div class="wrap">
            <h1><?php esc_html_e('DBack DB Tools', 'dback-db-tools'); ?></h1>
            <p><?php esc_html_e('Pure-PHP database tools for export, import, and SQL queries. No shell commands are used.', 'dback-db-tools'); ?></p>

            <div class="notice notice-info inline">
                <p>
                    <?php esc_html_e('Selected database:', 'dback-db-tools'); ?>
                    <code><?php echo esc_html(DB_NAME); ?></code>
                </p>
                <p>
                    <?php esc_html_e('REST API key:', 'dback-db-tools'); ?>
                    <code id="dback-api-key"><?php echo esc_html($api_key); ?></code>
                </p>
                <form method="post" style="margin-top:8px;">
                    <?php wp_nonce_field('dback_regenerate_key'); ?>
                    <button type="submit" name="dback_regenerate_key" class="button">
                        <?php esc_html_e('Regenerate API Key', 'dback-db-tools'); ?>
                    </button>
                </form>
            </div>

            <hr />

            <h2><?php esc_html_e('Export', 'dback-db-tools'); ?></h2>
            <p><?php esc_html_e('Download a gzip-compressed SQL dump of the current WordPress database.', 'dback-db-tools'); ?></p>
            <p>
                <button type="button" class="button button-primary" id="dback-export-button">
                    <?php esc_html_e('Export Database', 'dback-db-tools'); ?>
                </button>
            </p>
            <div id="dback-export-status" class="notice inline" style="display:none;"></div>

            <hr />

            <h2><?php esc_html_e('Import', 'dback-db-tools'); ?></h2>
            <p><?php esc_html_e('Upload a .sql.gz backup file to restore into the current database.', 'dback-db-tools'); ?></p>
            <table class="form-table" role="presentation">
                <tr>
                    <th scope="row">
                        <label for="dback-import-file"><?php esc_html_e('Backup file', 'dback-db-tools'); ?></label>
                    </th>
                    <td>
                        <input type="file" id="dback-import-file" accept=".gz,.sql.gz,application/gzip,application/x-gzip" />
                    </td>
                </tr>
            </table>
            <p>
                <button type="button" class="button button-primary" id="dback-import-button">
                    <?php esc_html_e('Import Database', 'dback-db-tools'); ?>
                </button>
            </p>
            <div id="dback-import-status" class="notice inline" style="display:none;"></div>

            <hr />

            <h2><?php esc_html_e('Run Query', 'dback-db-tools'); ?></h2>
            <p><?php esc_html_e('Execute SQL against the selected database.', 'dback-db-tools'); ?></p>
            <table class="form-table" role="presentation">
                <tr>
                    <th scope="row">
                        <label for="dback-query-sql"><?php esc_html_e('SQL', 'dback-db-tools'); ?></label>
                    </th>
                    <td>
                        <textarea id="dback-query-sql" class="large-text code" rows="8" placeholder="SELECT * FROM wp_posts LIMIT 10;"></textarea>
                    </td>
                </tr>
            </table>
            <p>
                <button type="button" class="button button-primary" id="dback-query-button">
                    <?php esc_html_e('Run Query', 'dback-db-tools'); ?>
                </button>
            </p>
            <div id="dback-query-status" class="notice inline" style="display:none;"></div>
            <div id="dback-query-result"></div>

            <hr />

            <h2><?php esc_html_e('Error Log', 'dback-db-tools'); ?></h2>
            <p><?php esc_html_e('Recent plugin errors from export, import, and query operations.', 'dback-db-tools'); ?></p>
            <p>
                <button type="button" class="button" id="dback-logs-refresh">
                    <?php esc_html_e('Refresh Log', 'dback-db-tools'); ?>
                </button>
                <button type="button" class="button" id="dback-logs-clear">
                    <?php esc_html_e('Clear Log', 'dback-db-tools'); ?>
                </button>
            </p>
            <div id="dback-logs-status" class="notice inline" style="display:none;"></div>
            <table class="widefat striped" id="dback-logs-table">
                <thead>
                    <tr>
                        <th scope="col"><?php esc_html_e('Time (UTC)', 'dback-db-tools'); ?></th>
                        <th scope="col"><?php esc_html_e('Operation', 'dback-db-tools'); ?></th>
                        <th scope="col"><?php esc_html_e('Code', 'dback-db-tools'); ?></th>
                        <th scope="col"><?php esc_html_e('Message', 'dback-db-tools'); ?></th>
                        <th scope="col"><?php esc_html_e('ID', 'dback-db-tools'); ?></th>
                    </tr>
                </thead>
                <tbody id="dback-logs-body">
                    <tr>
                        <td colspan="5"><?php esc_html_e('Loading...', 'dback-db-tools'); ?></td>
                    </tr>
                </tbody>
            </table>
        </div>
        <?php
    }
}
