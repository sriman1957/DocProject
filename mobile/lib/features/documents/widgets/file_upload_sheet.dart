import 'package:flutter/material.dart';

import '../controllers/document_upload_controller.dart';
import '../models/vault_document.dart';

enum UploadDestination { personalVault, subgroup }

class FileUploadSheet extends StatefulWidget {
  const FileUploadSheet({
    required this.uploadController,
    this.destination = UploadDestination.personalVault,
    this.groupId,
    this.subgroupId,
    this.subgroupName,
    this.vaultDocuments = const [],
    this.onSuccess,
    super.key,
  });

  final DocumentUploadController uploadController;
  final UploadDestination destination;
  final int? groupId;
  final int? subgroupId;
  final String? subgroupName;
  final List<VaultDocument> vaultDocuments;
  final VoidCallback? onSuccess;

  @override
  State<FileUploadSheet> createState() => _FileUploadSheetState();
}

class _FileUploadSheetState extends State<FileUploadSheet> {
  int? _selectedVaultDocId;
  bool _useExistingVaultDoc = false;

  @override
  void initState() {
    super.initState();
    widget.uploadController.reset();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    final isSubgroup = widget.destination == UploadDestination.subgroup;

    return Padding(
      padding: EdgeInsets.only(
        bottom: MediaQuery.of(context).viewInsets.bottom,
      ),
      child: Container(
        padding: const EdgeInsets.fromLTRB(24, 20, 24, 32),
        decoration: BoxDecoration(
          color: colors.surface,
          borderRadius: const BorderRadius.vertical(top: Radius.circular(28)),
        ),
        child: ListenableBuilder(
          listenable: widget.uploadController,
          builder: (context, _) {
            final state = widget.uploadController.state;

            if (state is UploadSuccess) {
              return _buildSuccessView(context, state.message);
            }

            return Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                // Drag handle
                Center(
                  child: Container(
                    width: 36,
                    height: 4,
                    margin: const EdgeInsets.only(bottom: 16),
                    decoration: BoxDecoration(
                      color: colors.outlineVariant,
                      borderRadius: BorderRadius.circular(2),
                    ),
                  ),
                ),

                // Sheet Header
                Row(
                  children: [
                    Icon(
                      isSubgroup
                          ? Icons.send_rounded
                          : Icons.upload_file_rounded,
                      color: colors.primary,
                      size: 26,
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            isSubgroup
                                ? 'Submit Document'
                                : 'Upload to Personal Vault',
                            style: theme.textTheme.titleMedium?.copyWith(
                              fontWeight: FontWeight.bold,
                            ),
                          ),
                          if (isSubgroup && widget.subgroupName != null)
                            Text(
                              'Destination: ${widget.subgroupName}',
                              style: theme.textTheme.bodySmall?.copyWith(
                                color: colors.onSurfaceVariant,
                              ),
                            ),
                        ],
                      ),
                    ),
                    if (!widget.uploadController.isUploading)
                      IconButton(
                        icon: const Icon(Icons.close_rounded),
                        onPressed: () => Navigator.of(context).pop(),
                        tooltip: 'Close',
                      ),
                  ],
                ),
                const SizedBox(height: 16),

                // Subgroup source selector toggle (if vault documents exist)
                if (isSubgroup && widget.vaultDocuments.isNotEmpty) ...[
                  SegmentedButton<bool>(
                    segments: const [
                      ButtonSegment(
                        value: false,
                        label: Text('From Device'),
                        icon: Icon(Icons.phone_android_rounded),
                      ),
                      ButtonSegment(
                        value: true,
                        label: Text('From Vault'),
                        icon: Icon(Icons.folder_shared_rounded),
                      ),
                    ],
                    selected: {_useExistingVaultDoc},
                    onSelectionChanged: widget.uploadController.isUploading
                        ? null
                        : (selected) {
                            setState(() {
                              _useExistingVaultDoc = selected.first;
                            });
                          },
                  ),
                  const SizedBox(height: 16),
                ],

                // Content: Device File Selection OR Vault Document Picker
                if (_useExistingVaultDoc && isSubgroup)
                  _buildVaultDocPicker(context)
                else
                  _buildDeviceFilePicker(context),

                // Error message if any
                if (state is UploadFailure) ...[
                  const SizedBox(height: 14),
                  Container(
                    padding: const EdgeInsets.all(12),
                    decoration: BoxDecoration(
                      color: colors.errorContainer,
                      borderRadius: BorderRadius.circular(12),
                    ),
                    child: Row(
                      children: [
                        Icon(
                          Icons.error_outline_rounded,
                          color: colors.onErrorContainer,
                          size: 20,
                        ),
                        const SizedBox(width: 10),
                        Expanded(
                          child: Text(
                            state.message,
                            style: theme.textTheme.bodySmall?.copyWith(
                              color: colors.onErrorContainer,
                              fontWeight: FontWeight.w500,
                            ),
                          ),
                        ),
                      ],
                    ),
                  ),
                ],

                const SizedBox(height: 24),

                // Action buttons
                _buildActionButtons(context),
              ],
            );
          },
        ),
      ),
    );
  }

  Widget _buildDeviceFilePicker(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    final file = widget.uploadController.selectedFile;
    final state = widget.uploadController.state;
    final validationError = state is UploadFileSelected
        ? state.validationError
        : null;

    if (file == null) {
      return InkWell(
        onTap: widget.uploadController.isUploading
            ? null
            : () => widget.uploadController.pickFile(),
        borderRadius: BorderRadius.circular(16),
        child: Container(
          padding: const EdgeInsets.symmetric(vertical: 24, horizontal: 16),
          decoration: BoxDecoration(
            color: colors.surfaceContainerLow,
            borderRadius: BorderRadius.circular(16),
            border: Border.all(
              color: colors.outlineVariant,
              style: BorderStyle.solid,
            ),
          ),
          child: Column(
            children: [
              Container(
                width: 52,
                height: 52,
                decoration: BoxDecoration(
                  color: colors.primaryContainer.withValues(alpha: 0.6),
                  shape: BoxShape.circle,
                ),
                child: Icon(
                  Icons.cloud_upload_outlined,
                  size: 28,
                  color: colors.primary,
                ),
              ),
              const SizedBox(height: 12),
              Text(
                'Tap to select a document',
                style: theme.textTheme.titleSmall?.copyWith(
                  fontWeight: FontWeight.bold,
                ),
              ),
              const SizedBox(height: 4),
              Text(
                'Supported: PDF, DOCX, PNG, JPEG (Max 20 MB)',
                style: theme.textTheme.bodySmall?.copyWith(
                  color: colors.onSurfaceVariant,
                ),
              ),
            ],
          ),
        ),
      );
    }

    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: colors.surfaceContainerLow,
        borderRadius: BorderRadius.circular(14),
        border: Border.all(
          color: validationError != null
              ? colors.error
              : colors.outlineVariant.withValues(alpha: 0.5),
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Container(
                width: 44,
                height: 44,
                decoration: BoxDecoration(
                  color: colors.secondaryContainer.withValues(alpha: 0.7),
                  borderRadius: BorderRadius.circular(10),
                ),
                child: Icon(
                  Icons.insert_drive_file_rounded,
                  color: colors.onSecondaryContainer,
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      file.name,
                      style: theme.textTheme.bodyMedium?.copyWith(
                        fontWeight: FontWeight.bold,
                      ),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                    const SizedBox(height: 2),
                    Text(
                      file.formattedSize,
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: colors.onSurfaceVariant,
                      ),
                    ),
                  ],
                ),
              ),
              if (!widget.uploadController.isUploading) ...[
                IconButton(
                  icon: const Icon(Icons.change_circle_outlined),
                  tooltip: 'Change file',
                  onPressed: () => widget.uploadController.pickFile(),
                ),
                IconButton(
                  icon: const Icon(Icons.delete_outline_rounded),
                  tooltip: 'Remove',
                  onPressed: () => widget.uploadController.clearFile(),
                ),
              ],
            ],
          ),
          if (validationError != null) ...[
            const SizedBox(height: 10),
            Text(
              validationError,
              style: theme.textTheme.bodySmall?.copyWith(
                color: colors.error,
                fontWeight: FontWeight.w600,
              ),
            ),
          ],
        ],
      ),
    );
  }

  Widget _buildVaultDocPicker(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;

    return Container(
      constraints: const BoxConstraints(maxHeight: 200),
      decoration: BoxDecoration(
        color: colors.surfaceContainerLow,
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: colors.outlineVariant),
      ),
      child: Material(
        type: MaterialType.transparency,
        child: ListView.separated(
          shrinkWrap: true,
          itemCount: widget.vaultDocuments.length,
          separatorBuilder: (context, _) => const Divider(height: 1),
          itemBuilder: (context, index) {
            final doc = widget.vaultDocuments[index];
            final isSelected = _selectedVaultDocId == doc.id;

            return ListTile(
              dense: true,
              selected: isSelected,
              selectedTileColor: colors.primaryContainer.withValues(alpha: 0.3),
              leading: Icon(doc.fileIcon, color: doc.fileIconColor),
              title: Text(
                doc.originalFilename,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: theme.textTheme.bodyMedium?.copyWith(
                  fontWeight: isSelected ? FontWeight.bold : FontWeight.normal,
                ),
              ),
              subtitle: Text(
                '${doc.formattedFileSize} • ${doc.formattedUploadDate}',
                style: theme.textTheme.bodySmall?.copyWith(
                  color: colors.onSurfaceVariant,
                ),
              ),
              trailing: isSelected
                  ? Icon(Icons.check_circle_rounded, color: colors.primary)
                  : null,
              onTap: widget.uploadController.isUploading
                  ? null
                  : () {
                      setState(() {
                        _selectedVaultDocId = doc.id;
                      });
                    },
            );
          },
        ),
      ),
    );
  }

  Widget _buildActionButtons(BuildContext context) {
    final isUploading = widget.uploadController.isUploading;
    final file = widget.uploadController.selectedFile;
    final state = widget.uploadController.state;
    final hasValidationError =
        state is UploadFileSelected && state.validationError != null;

    final canSubmit =
        !isUploading &&
        ((_useExistingVaultDoc && _selectedVaultDocId != null) ||
            (!_useExistingVaultDoc && file != null && !hasValidationError));

    return Row(
      children: [
        Expanded(
          child: OutlinedButton(
            onPressed: isUploading ? null : () => Navigator.of(context).pop(),
            child: const Text('Cancel'),
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          flex: 2,
          child: FilledButton.icon(
            onPressed: canSubmit ? _handleSubmit : null,
            icon: isUploading
                ? const SizedBox(
                    width: 18,
                    height: 18,
                    child: CircularProgressIndicator(
                      strokeWidth: 2,
                      color: Colors.white,
                    ),
                  )
                : const Icon(Icons.upload_rounded, size: 20),
            label: Text(
              isUploading
                  ? 'Processing...'
                  : (widget.destination == UploadDestination.subgroup
                        ? 'Submit'
                        : 'Upload'),
            ),
          ),
        ),
      ],
    );
  }

  Future<void> _handleSubmit() async {
    VaultDocument? result;

    if (widget.destination == UploadDestination.personalVault) {
      result = await widget.uploadController.uploadToPersonalVault();
    } else {
      if (widget.groupId == null || widget.subgroupId == null) return;

      result = await widget.uploadController.submitToSubgroup(
        groupId: widget.groupId!,
        subgroupId: widget.subgroupId!,
        existingVaultDocId: _useExistingVaultDoc ? _selectedVaultDocId : null,
      );
    }

    if (result != null) {
      widget.onSuccess?.call();
    }
  }

  Widget _buildSuccessView(BuildContext context, String message) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 24),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Container(
            width: 64,
            height: 64,
            decoration: BoxDecoration(
              color: Colors.green.withValues(alpha: 0.15),
              shape: BoxShape.circle,
            ),
            child: const Icon(
              Icons.check_circle_rounded,
              size: 40,
              color: Colors.green,
            ),
          ),
          const SizedBox(height: 16),
          Text(
            'Success!',
            style: theme.textTheme.titleMedium?.copyWith(
              fontWeight: FontWeight.bold,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            message,
            textAlign: TextAlign.center,
            style: theme.textTheme.bodyMedium?.copyWith(
              color: colors.onSurfaceVariant,
            ),
          ),
          const SizedBox(height: 24),
          FilledButton(
            onPressed: () => Navigator.of(context).pop(),
            child: const Text('Done'),
          ),
        ],
      ),
    );
  }
}
