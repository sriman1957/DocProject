import 'package:flutter/material.dart';

import '../models/vault_document.dart';

class SubgroupDocumentCard extends StatelessWidget {
  const SubgroupDocumentCard({
    required this.document,
    required this.onDownload,
    this.isDownloading = false,
    this.onTap,
    super.key,
  });

  final VaultDocument document;
  final VoidCallback onDownload;
  final bool isDownloading;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    final iconColor = document.fileIconColor;

    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(color: colors.outlineVariant.withValues(alpha: 0.5)),
      ),
      color: colors.surfaceContainerLow,
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Container(
                    width: 48,
                    height: 48,
                    decoration: BoxDecoration(
                      color: iconColor.withValues(alpha: 0.12),
                      borderRadius: BorderRadius.circular(12),
                      border: Border.all(
                        color: iconColor.withValues(alpha: 0.25),
                      ),
                    ),
                    child: Icon(document.fileIcon, color: iconColor, size: 26),
                  ),
                  const SizedBox(width: 14),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          document.originalFilename,
                          style: theme.textTheme.titleMedium?.copyWith(
                            fontWeight: FontWeight.bold,
                          ),
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                        ),
                        const SizedBox(height: 6),
                        Wrap(
                          spacing: 8,
                          runSpacing: 4,
                          crossAxisAlignment: WrapCrossAlignment.center,
                          children: [
                            if (document.fileExtension.isNotEmpty)
                              Container(
                                padding: const EdgeInsets.symmetric(
                                  horizontal: 6,
                                  vertical: 2,
                                ),
                                decoration: BoxDecoration(
                                  color: colors.surfaceContainerHighest,
                                  borderRadius: BorderRadius.circular(4),
                                ),
                                child: Text(
                                  document.fileExtension,
                                  style: theme.textTheme.labelSmall?.copyWith(
                                    fontWeight: FontWeight.bold,
                                    color: colors.onSurfaceVariant,
                                  ),
                                ),
                              ),
                            Row(
                              mainAxisSize: MainAxisSize.min,
                              children: [
                                Icon(
                                  Icons.data_usage_rounded,
                                  size: 14,
                                  color: colors.onSurfaceVariant,
                                ),
                                const SizedBox(width: 4),
                                Text(
                                  document.formattedFileSize,
                                  style: theme.textTheme.bodySmall?.copyWith(
                                    color: colors.onSurfaceVariant,
                                  ),
                                ),
                              ],
                            ),
                            Row(
                              mainAxisSize: MainAxisSize.min,
                              children: [
                                Icon(
                                  Icons.schedule_rounded,
                                  size: 14,
                                  color: colors.onSurfaceVariant,
                                ),
                                const SizedBox(width: 4),
                                Text(
                                  document.formattedUploadDate,
                                  style: theme.textTheme.bodySmall?.copyWith(
                                    color: colors.onSurfaceVariant,
                                  ),
                                ),
                              ],
                            ),
                          ],
                        ),
                      ],
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 12),

              // Integrity & Document ID badge
              Container(
                width: double.infinity,
                padding: const EdgeInsets.symmetric(
                  horizontal: 10,
                  vertical: 6,
                ),
                decoration: BoxDecoration(
                  color: colors.surfaceContainerHighest.withValues(alpha: 0.4),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Row(
                  children: [
                    Icon(
                      Icons.verified_user_outlined,
                      size: 14,
                      color: colors.primary,
                    ),
                    const SizedBox(width: 6),
                    Expanded(
                      child: Text(
                        document.sha256.isNotEmpty
                            ? 'SHA-256: ${document.sha256.length > 16 ? "${document.sha256.substring(0, 16)}..." : document.sha256}'
                            : 'MIME: ${document.mimeType}',
                        style: theme.textTheme.labelSmall?.copyWith(
                          fontFamily: 'monospace',
                          color: colors.onSurfaceVariant,
                        ),
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                      ),
                    ),
                    Text(
                      '#${document.id}',
                      style: theme.textTheme.labelSmall?.copyWith(
                        color: colors.onSurfaceVariant,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ],
                ),
              ),

              const SizedBox(height: 12),

              // Action row with Download button
              Align(
                alignment: Alignment.centerRight,
                child: FilledButton.tonalIcon(
                  key: Key('download_button_${document.id}'),
                  onPressed: isDownloading ? null : onDownload,
                  icon: isDownloading
                      ? SizedBox(
                          width: 16,
                          height: 16,
                          child: CircularProgressIndicator(
                            strokeWidth: 2,
                            color: colors.primary,
                          ),
                        )
                      : const Icon(Icons.download_rounded, size: 18),
                  label: Text(isDownloading ? 'Downloading...' : 'Download'),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
