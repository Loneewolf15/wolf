<?php

/**
 * Image Processing Service
 * 
 * Handles image manipulation (resize, compress, thumbnails)
 * Only used if image_processing['enabled'] = 1 in config
 */
class ImageProcessor
{
    /**
     * Resize image to specified dimensions
     * 
     * @param string $sourcePath Original image path
     * @param int $width Target width
     * @param int $height Target height
     * @param string $variant Variant name (thumbnail, small, medium)
     * @return string Path to resized image
     */
    public function resize(string $sourcePath, int $width, int $height, string $variant = 'resized'): string
    {
        // Get image info
        $imageInfo = getimagesize($sourcePath);
        if (!$imageInfo) {
            throw new Exception('Invalid image file');
        }

        list($origWidth, $origHeight, $imageType) = $imageInfo;

        // Create image resource from source
        $sourceImage = $this->createImageFromType($sourcePath, $imageType);

        // Calculate aspect ratio
        $aspectRatio = $origWidth / $origHeight;
        $targetAspectRatio = $width / $height;

        // Adjust dimensions to maintain aspect ratio
        if ($aspectRatio > $targetAspectRatio) {
            // Image is wider
            $newWidth = $width;
            $newHeight = intval($width / $aspectRatio);
        } else {
            // Image is taller
            $newHeight = $height;
            $newWidth = intval($height * $aspectRatio);
        }

        // Create blank canvas
        $resizedImage = imagecreatetruecolor($newWidth, $newHeight);

        // Preserve transparency for PNG
        if ($imageType === IMAGETYPE_PNG) {
            imagealphablending($resizedImage, false);
            imagesavealpha($resizedImage, true);
            $transparent = imagecolorallocatealpha($resizedImage, 255, 255, 255, 127);
            imagefilledrectangle($resizedImage, 0, 0, $newWidth, $newHeight, $transparent);
        }

        // Resize
        imagecopyresampled(
            $resizedImage,
            $sourceImage,
            0,
            0,
            0,
            0,
            $newWidth,
            $newHeight,
            $origWidth,
            $origHeight
        );

        // Generate output path
        $pathInfo = pathinfo($sourcePath);
        $outputPath = $pathInfo['dirname'] . '/' . $pathInfo['filename'] . "_{$variant}." . $pathInfo['extension'];

        // Save image
        $this->saveImageByType($resizedImage, $outputPath, $imageType);

        // Free memory
        imagedestroy($sourceImage);
        imagedestroy($resizedImage);

        return $outputPath;
    }

    /**
     * Compress image
     */
    public function compress(string $sourcePath, int $quality = 85): bool
    {
        $imageInfo = getimagesize($sourcePath);
        if (!$imageInfo) {
            return false;
        }

        list($width, $height, $imageType) = $imageInfo;

        $sourceImage = $this->createImageFromType($sourcePath, $imageType);

        // Save with compression
        $this->saveImageByType($sourceImage, $sourcePath, $imageType, $quality);

        imagedestroy($sourceImage);

        return true;
    }

    /**
     * Generate multiple thumbnail sizes
     */
    public function generateThumbnails(string $sourcePath, array $sizes): array
    {
        $thumbnails = [];

        foreach ($sizes as $name => $dimensions) {
            try {
                $path = $this->resize(
                    $sourcePath,
                    $dimensions['width'],
                    $dimensions['height'],
                    $name
                );
                $thumbnails[$name] = $path;
            } catch (Exception $e) {
                error_log("Thumbnail generation failed for {$name}: " . $e->getMessage());
            }
        }

        return $thumbnails;
    }

    /**
     * Create image resource from file based on type
     */
    private function createImageFromType(string $path, int $imageType)
    {
        switch ($imageType) {
            case IMAGETYPE_JPEG:
                return imagecreatefromjpeg($path);
            case IMAGETYPE_PNG:
                return imagecreatefrompng($path);
            case IMAGETYPE_GIF:
                return imagecreatefromgif($path);
            case IMAGETYPE_WEBP:
                return imagecreatefromwebp($path);
            default:
                throw new Exception('Unsupported image type');
        }
    }

    /**
     * Save image based on type
     */
    private function saveImageByType($image, string $path, int $imageType, int $quality = 85): bool
    {
        switch ($imageType) {
            case IMAGETYPE_JPEG:
                return imagejpeg($image, $path, $quality);
            case IMAGETYPE_PNG:
                // PNG quality is 0-9 (compression level)
                $pngQuality = intval((100 - $quality) / 11);
                return imagepng($image, $path, $pngQuality);
            case IMAGETYPE_GIF:
                return imagegif($image, $path);
            case IMAGETYPE_WEBP:
                return imagewebp($image, $path, $quality);
            default:
                return false;
        }
    }
}
