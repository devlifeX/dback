<?php

if (!defined('ABSPATH')) {
    exit;
}

class DBack_Gzip_Stream {

    /** @var resource|null */
    private $output = null;

    /** @var DeflateContext|resource|null */
    private $context = null;

    /**
     * @param string $filename
     * @throws RuntimeException
     */
    public function __construct($filename = 'php://output') {
        if (!function_exists('deflate_init')) {
            throw new RuntimeException('The zlib extension is required for gzip export.');
        }

        $this->output = fopen($filename, 'wb');
        if (false === $this->output) {
            throw new RuntimeException('Unable to open gzip output stream.');
        }

        $this->context = deflate_init(ZLIB_ENCODING_GZIP, array('level' => 9));
        if (false === $this->context) {
            fclose($this->output);
            $this->output = null;
            throw new RuntimeException('Unable to initialize gzip compression.');
        }
    }

    /**
     * @param string $data
     */
    public function write($data) {
        $compressed = deflate_add($this->context, $data, ZLIB_NO_FLUSH);
        if (false === $compressed) {
            throw new RuntimeException('Gzip compression failed.');
        }

        $written = fwrite($this->output, $compressed);
        if (false === $written) {
            throw new RuntimeException('Unable to write to output stream.');
        }

        fflush($this->output);
        if (function_exists('flush')) {
            flush();
        }
    }

    /**
     * @param string $line
     */
    public function write_line($line) {
        $this->write($line . "\n");
    }

    public function close() {
        if (null === $this->output) {
            return;
        }

        $final = deflate_add($this->context, '', ZLIB_FINISH);
        if (false !== $final) {
            fwrite($this->output, $final);
        }

        fclose($this->output);
        $this->output = null;
        $this->context = null;
    }
}
