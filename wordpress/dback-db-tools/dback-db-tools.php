<?php
/**
 * Plugin Name: DBack DB Tools
 * Description: Pure-PHP database export, import, and SQL query tools for DBack. No shell commands required.
 * Version: 1.0.0
 * Author: DBack
 * Requires PHP: 7.4
 * Requires at least: 5.8
 */

if (!defined('ABSPATH')) {
    exit;
}

define('DBACK_DB_TOOLS_VERSION', '1.0.0');
define('DBACK_DB_TOOLS_FILE', __FILE__);
define('DBACK_DB_TOOLS_PATH', plugin_dir_path(__FILE__));
define('DBACK_DB_TOOLS_URL', plugin_dir_url(__FILE__));
define('DBACK_DB_TOOLS_REST_NAMESPACE', 'dback/v1');
define('DBACK_HARDCODED_API_KEY', '{{DBACK_API_KEY}}');

require_once DBACK_DB_TOOLS_PATH . 'includes/class-dback-api-key.php';
require_once DBACK_DB_TOOLS_PATH . 'includes/class-dback-database.php';
require_once DBACK_DB_TOOLS_PATH . 'includes/class-dback-error-logger.php';
require_once DBACK_DB_TOOLS_PATH . 'includes/class-dback-gzip-stream.php';
require_once DBACK_DB_TOOLS_PATH . 'vendor/ifsnop/mysqldump-php/src/Ifsnop/Mysqldump/Mysqldump.php';
require_once DBACK_DB_TOOLS_PATH . 'includes/class-dback-exporter-mysqli.php';
require_once DBACK_DB_TOOLS_PATH . 'includes/class-dback-exporter.php';
require_once DBACK_DB_TOOLS_PATH . 'includes/class-dback-importer.php';
require_once DBACK_DB_TOOLS_PATH . 'includes/class-dback-query-runner.php';
require_once DBACK_DB_TOOLS_PATH . 'includes/class-dback-preflight.php';
require_once DBACK_DB_TOOLS_PATH . 'includes/class-dback-rest-controller.php';
require_once DBACK_DB_TOOLS_PATH . 'includes/class-dback-admin-page.php';

register_activation_hook(__FILE__, array('DBack_Api_Key', 'activate'));

final class DBack_DB_Tools_Plugin {

    /** @var self|null */
    private static $instance = null;

    /** @var DBack_Rest_Controller */
    private $rest_controller;

    /** @var DBack_Admin_Page */
    private $admin_page;

    public static function instance() {
        if (null === self::$instance) {
            self::$instance = new self();
        }

        return self::$instance;
    }

    private function __construct() {
        $this->rest_controller = new DBack_Rest_Controller();
        $this->admin_page = new DBack_Admin_Page();

        add_action('rest_api_init', array($this->rest_controller, 'register_routes'));
        add_action('admin_menu', array($this->admin_page, 'register_menu'));
        add_action('admin_enqueue_scripts', array($this->admin_page, 'enqueue_assets'));
    }
}

DBack_DB_Tools_Plugin::instance();
