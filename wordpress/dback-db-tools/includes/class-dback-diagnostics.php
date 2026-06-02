<?php

if (!defined('ABSPATH')) {
    exit;
}

class DBack_Diagnostics {

    /** @var string[] */
    private static $expected_routes = array(
        'GET /ping',
        'GET /preflight',
        'GET /export',
        'POST /import',
        'POST /query',
        'GET /logs',
        'DELETE /logs',
    );

    /**
     * @return array<string,mixed>
     */
    public static function collect_report() {
        $plugin_basename = plugin_basename(DBACK_DB_TOOLS_FILE);
        $registered = self::registered_route_map();
        $route_tests = self::test_internal_routes();
        $rest_disabled = self::rest_api_disabled_reason();
        $issues = self::detect_issues($registered, $route_tests, $rest_disabled);

        return array(
            'generated_at' => gmdate('c'),
            'plugin_version' => DBACK_DB_TOOLS_VERSION,
            'plugin_basename' => $plugin_basename,
            'plugin_folder' => basename(dirname(DBACK_DB_TOOLS_FILE)),
            'plugin_active' => is_plugin_active($plugin_basename),
            'hardcoded_key_configured' => '' !== DBack_Api_Key::hardcoded_key(),
            'stored_key_present' => '' !== DBack_Api_Key::get(),
            'auth_mode' => self::auth_mode_label(),
            'rest_namespace' => DBACK_DB_TOOLS_REST_NAMESPACE,
            'rest_index_url' => rest_url(),
            'rest_namespace_url' => untrailingslashit(rest_url(DBACK_DB_TOOLS_REST_NAMESPACE)),
            'endpoint_urls' => self::endpoint_urls(),
            'permalink_structure' => (string) get_option('permalink_structure'),
            'pretty_permalinks' => (bool) get_option('permalink_structure'),
            'site_url' => site_url(),
            'home_url' => home_url(),
            'wordpress_version' => get_bloginfo('version'),
            'php_version' => PHP_VERSION,
            'driver' => class_exists('DBack_Database') ? DBack_Database::driver() : 'unknown',
            'rest_api_disabled_reason' => $rest_disabled,
            'registered_routes' => $registered,
            'expected_routes' => self::$expected_routes,
            'route_tests' => $route_tests,
            'issues' => $issues,
            'log_file' => basename(DBack_Error_Logger::log_file_path()),
            'log_file_writable' => is_writable(dirname(DBack_Error_Logger::log_file_path())),
        );
    }

    /**
     * @return string
     */
    private static function auth_mode_label() {
        if ('' !== DBack_Api_Key::hardcoded_key()) {
            return 'hardcoded_key';
        }
        if ('' !== DBack_Api_Key::get()) {
            return 'wordpress_option';
        }

        return 'none';
    }

    /**
     * @return array<string,string>
     */
    private static function endpoint_urls() {
        $base = untrailingslashit(rest_url(DBACK_DB_TOOLS_REST_NAMESPACE));

        return array(
            'ping' => $base . '/ping',
            'preflight' => $base . '/preflight',
            'export' => $base . '/export',
            'import' => $base . '/import',
            'query' => $base . '/query',
            'logs' => $base . '/logs',
        );
    }

    /**
     * @return string
     */
    private static function rest_api_disabled_reason() {
        if (has_filter('rest_authentication_errors')) {
            // Filter exists; actual behavior depends on callback.
        }

        if (defined('REST_API_DISABLED') && REST_API_DISABLED) {
            return 'REST_API_DISABLED constant is true.';
        }

        /** @var mixed $errors */
        $errors = apply_filters('rest_authentication_errors', null);
        if (is_wp_error($errors) && 'rest_disabled' === $errors->get_error_code()) {
            return $errors->get_error_message();
        }

        return '';
    }

    /**
     * @return array<int,array<string,mixed>>
     */
    private static function registered_route_map() {
        if (!function_exists('rest_get_server')) {
            return array();
        }

        $server = rest_get_server();
        $routes = $server->get_routes();
        $prefix = '/' . trim(DBACK_DB_TOOLS_REST_NAMESPACE, '/');
        $found = array();

        foreach ($routes as $path => $handlers) {
            if (0 !== strpos($path, $prefix)) {
                continue;
            }

            $methods = array();
            foreach ((array) $handlers as $handler) {
                if (!empty($handler['methods']) && is_array($handler['methods'])) {
                    foreach (array_keys($handler['methods']) as $method) {
                        $methods[] = strtoupper($method);
                    }
                }
            }

            $methods = array_values(array_unique($methods));
            sort($methods);

            $found[] = array(
                'path' => $path,
                'methods' => $methods,
            );
        }

        return $found;
    }

    /**
     * @return array<string,array<string,mixed>>
     */
    private static function test_internal_routes() {
        $tests = array();

        foreach (array(
            'ping' => 'GET',
            'preflight' => 'GET',
        ) as $route => $method) {
            $tests[$route] = self::dispatch_route($method, '/' . DBACK_DB_TOOLS_REST_NAMESPACE . '/' . $route);
        }

        return $tests;
    }

    /**
     * @param string $method
     * @param string $route
     * @return array<string,mixed>
     */
    private static function dispatch_route($method, $route) {
        if (!class_exists('WP_REST_Request')) {
            return array(
                'ok' => false,
                'status' => 0,
                'code' => 'missing_rest',
                'message' => 'WordPress REST classes are unavailable.',
            );
        }

        $request = new WP_REST_Request($method, $route);
        $response = rest_do_request($request);

        if ($response->is_error()) {
            $error = $response->as_error();

            return array(
                'ok' => 'rest_no_route' !== $error->get_error_code(),
                'status' => (int) $response->get_status(),
                'code' => $error->get_error_code(),
                'message' => $error->get_error_message(),
            );
        }

        $data = $response->get_data();

        return array(
            'ok' => true,
            'status' => (int) $response->get_status(),
            'code' => 'ok',
            'message' => is_array($data) && !empty($data['message']) ? (string) $data['message'] : 'Route responded.',
        );
    }

    /**
     * @param array<int,array<string,mixed>> $registered
     * @param array<string,array<string,mixed>> $route_tests
     * @param string $rest_disabled
     * @return array<int,array<string,string>>
     */
    private static function detect_issues($registered, $route_tests, $rest_disabled) {
        $issues = array();

        if (!is_plugin_active(plugin_basename(DBACK_DB_TOOLS_FILE))) {
            $issues[] = array(
                'level' => 'error',
                'message' => __('DBack DB Tools is installed but not activated. Activate the plugin to register REST routes.', 'dback-db-tools'),
            );
        }

        if ('' !== $rest_disabled) {
            $issues[] = array(
                'level' => 'error',
                'message' => sprintf(
                    /* translators: %s: reason REST is disabled */
                    __('WordPress REST API appears disabled: %s', 'dback-db-tools'),
                    $rest_disabled
                ),
            );
        }

        if (!get_option('permalink_structure')) {
            $issues[] = array(
                'level' => 'warning',
                'message' => __('Pretty permalinks are not enabled. Some hosts require them for /wp-json/ routes. Try Settings → Permalinks → Post name.', 'dback-db-tools'),
            );
        }

        if ('none' === self::auth_mode_label()) {
            $issues[] = array(
                'level' => 'warning',
                'message' => __('No API key is configured. Download the plugin from DBack with a token, or regenerate a key below.', 'dback-db-tools'),
            );
        }

        if (empty($registered)) {
            $issues[] = array(
                'level' => 'error',
                'message' => __('No dback/v1 REST routes are registered. This causes rest_no_route errors in DBack.', 'dback-db-tools'),
            );
        }

        foreach ($route_tests as $route => $result) {
            if (empty($result['ok']) && isset($result['code']) && 'rest_no_route' === $result['code']) {
                $issues[] = array(
                    'level' => 'error',
                    'message' => sprintf(
                        /* translators: 1: route name */
                        __('Internal test for /%1$s failed with rest_no_route. The route is not registered.', 'dback-db-tools'),
                        DBACK_DB_TOOLS_REST_NAMESPACE . '/' . $route
                    ),
                );
            }
        }

        return $issues;
    }

    /**
     * @param array<string,mixed> $report
     */
    public static function log_report_snapshot($report) {
        DBack_Error_Logger::log(
            'info',
            'dback_diagnostics_snapshot',
            'Diagnostics snapshot generated.',
            array(
                'operation' => 'diagnostics',
                'plugin_active' => !empty($report['plugin_active']),
                'routes_registered' => count($report['registered_routes']),
                'issues' => count($report['issues']),
            )
        );
    }

    /**
     * @param array<string,mixed> $report
     */
    public static function render_admin_section($report) {
        $issues = isset($report['issues']) && is_array($report['issues']) ? $report['issues'] : array();
        ?>
        <h2 id="dback-diagnostics"><?php esc_html_e('Status & Diagnostics', 'dback-db-tools'); ?></h2>
        <p><?php esc_html_e('Use this section when DBack reports rest_no_route or other WordPress API errors.', 'dback-db-tools'); ?></p>

        <?php if (!empty($issues)) : ?>
            <?php foreach ($issues as $issue) : ?>
                <?php
                $level = isset($issue['level']) ? $issue['level'] : 'warning';
                $class = 'error' === $level ? 'notice-error' : 'notice-warning';
                ?>
                <div class="notice <?php echo esc_attr($class); ?> inline">
                    <p><?php echo esc_html($issue['message']); ?></p>
                </div>
            <?php endforeach; ?>
        <?php else : ?>
            <div class="notice notice-success inline">
                <p><?php esc_html_e('No blocking issues detected. REST routes appear registered.', 'dback-db-tools'); ?></p>
            </div>
        <?php endif; ?>

        <table class="widefat striped" style="max-width:960px;">
            <tbody>
                <tr>
                    <th scope="row"><?php esc_html_e('Plugin version', 'dback-db-tools'); ?></th>
                    <td><code><?php echo esc_html($report['plugin_version']); ?></code></td>
                </tr>
                <tr>
                    <th scope="row"><?php esc_html_e('Plugin folder', 'dback-db-tools'); ?></th>
                    <td><code><?php echo esc_html($report['plugin_folder']); ?></code></td>
                </tr>
                <tr>
                    <th scope="row"><?php esc_html_e('Plugin active', 'dback-db-tools'); ?></th>
                    <td><?php echo !empty($report['plugin_active']) ? esc_html__('Yes', 'dback-db-tools') : esc_html__('No', 'dback-db-tools'); ?></td>
                </tr>
                <tr>
                    <th scope="row"><?php esc_html_e('Auth mode', 'dback-db-tools'); ?></th>
                    <td><code><?php echo esc_html($report['auth_mode']); ?></code></td>
                </tr>
                <tr>
                    <th scope="row"><?php esc_html_e('Hardcoded DBack token', 'dback-db-tools'); ?></th>
                    <td><?php echo !empty($report['hardcoded_key_configured']) ? esc_html__('Configured', 'dback-db-tools') : esc_html__('Missing (download plugin from DBack)', 'dback-db-tools'); ?></td>
                </tr>
                <tr>
                    <th scope="row"><?php esc_html_e('REST index', 'dback-db-tools'); ?></th>
                    <td><code><?php echo esc_html($report['rest_index_url']); ?></code></td>
                </tr>
                <tr>
                    <th scope="row"><?php esc_html_e('REST namespace URL', 'dback-db-tools'); ?></th>
                    <td><code><?php echo esc_html($report['rest_namespace_url']); ?></code></td>
                </tr>
                <tr>
                    <th scope="row"><?php esc_html_e('DBack ping URL', 'dback-db-tools'); ?></th>
                    <td><code><?php echo esc_html($report['endpoint_urls']['ping']); ?></code></td>
                </tr>
                <tr>
                    <th scope="row"><?php esc_html_e('Permalink structure', 'dback-db-tools'); ?></th>
                    <td>
                        <?php if (!empty($report['pretty_permalinks'])) : ?>
                            <code><?php echo esc_html($report['permalink_structure']); ?></code>
                        <?php else : ?>
                            <span><?php esc_html_e('Plain (not recommended for REST)', 'dback-db-tools'); ?></span>
                        <?php endif; ?>
                    </td>
                </tr>
                <tr>
                    <th scope="row"><?php esc_html_e('Site URL', 'dback-db-tools'); ?></th>
                    <td><code><?php echo esc_html($report['site_url']); ?></code></td>
                </tr>
                <tr>
                    <th scope="row"><?php esc_html_e('Registered dback routes', 'dback-db-tools'); ?></th>
                    <td><?php echo esc_html((string) count($report['registered_routes'])); ?></td>
                </tr>
                <tr>
                    <th scope="row"><?php esc_html_e('Internal route test', 'dback-db-tools'); ?></th>
                    <td>
                        <?php foreach ($report['route_tests'] as $route => $result) : ?>
                            <div>
                                <code><?php echo esc_html('/' . DBACK_DB_TOOLS_REST_NAMESPACE . '/' . $route); ?></code>
                                —
                                <?php
                                echo esc_html(
                                    sprintf(
                                        '%s (%s)',
                                        isset($result['code']) ? $result['code'] : 'unknown',
                                        isset($result['message']) ? $result['message'] : ''
                                    )
                                );
                                ?>
                            </div>
                        <?php endforeach; ?>
                    </td>
                </tr>
                <tr>
                    <th scope="row"><?php esc_html_e('Debug log file', 'dback-db-tools'); ?></th>
                    <td>
                        <code><?php echo esc_html($report['log_file']); ?></code>
                        <?php if (!empty($report['log_file_writable'])) : ?>
                            <?php esc_html_e('(writable)', 'dback-db-tools'); ?>
                        <?php else : ?>
                            <?php esc_html_e('(not writable)', 'dback-db-tools'); ?>
                        <?php endif; ?>
                    </td>
                </tr>
            </tbody>
        </table>

        <?php if (!empty($report['registered_routes'])) : ?>
            <h3><?php esc_html_e('Registered routes', 'dback-db-tools'); ?></h3>
            <table class="widefat striped" style="max-width:960px;">
                <thead>
                    <tr>
                        <th scope="col"><?php esc_html_e('Path', 'dback-db-tools'); ?></th>
                        <th scope="col"><?php esc_html_e('Methods', 'dback-db-tools'); ?></th>
                    </tr>
                </thead>
                <tbody>
                    <?php foreach ($report['registered_routes'] as $route) : ?>
                        <tr>
                            <td><code><?php echo esc_html($route['path']); ?></code></td>
                            <td><code><?php echo esc_html(implode(', ', $route['methods'])); ?></code></td>
                        </tr>
                    <?php endforeach; ?>
                </tbody>
            </table>
        <?php endif; ?>
        <?php
    }

    public static function register_plugin_links() {
        add_filter(
            'plugin_action_links_' . plugin_basename(DBACK_DB_TOOLS_FILE),
            array(__CLASS__, 'plugin_action_links')
        );
        add_filter('plugin_row_meta', array(__CLASS__, 'plugin_row_meta'), 10, 2);
    }

    public static function register_admin_notices() {
        add_action('admin_notices', array(__CLASS__, 'render_admin_notice'));
    }

    public static function render_admin_notice() {
        if (!current_user_can('manage_options')) {
            return;
        }

        $basename = plugin_basename(DBACK_DB_TOOLS_FILE);
        if (!is_plugin_active($basename)) {
            return;
        }

        $registered = self::registered_route_map();
        if (!empty($registered)) {
            return;
        }

        $url = admin_url('tools.php?page=dback-db-tools#dback-diagnostics');
        ?>
        <div class="notice notice-error">
            <p>
                <?php
                echo wp_kses_post(
                    sprintf(
                        /* translators: %s: diagnostics page URL */
                        __('<strong>DBack DB Tools:</strong> REST routes are not registered. DBack cannot connect until this is fixed. <a href="%s">Open Status &amp; Diagnostics</a>', 'dback-db-tools'),
                        esc_url($url)
                    )
                );
                ?>
            </p>
        </div>
        <?php
    }

    /**
     * @param string[] $links
     * @return string[]
     */
    public static function plugin_action_links($links) {
        $url = admin_url('tools.php?page=dback-db-tools#dback-diagnostics');
        $links[] = '<a href="' . esc_url($url) . '">' . esc_html__('Status & Logs', 'dback-db-tools') . '</a>';

        return $links;
    }

    /**
     * @param string[] $links
     * @param string $file
     * @return string[]
     */
    public static function plugin_row_meta($links, $file) {
        if (plugin_basename(DBACK_DB_TOOLS_FILE) !== $file) {
            return $links;
        }

        $links[] = '<a href="' . esc_url(admin_url('tools.php?page=dback-db-tools#dback-diagnostics')) . '">' . esc_html__('Diagnostics', 'dback-db-tools') . '</a>';

        return $links;
    }
}
