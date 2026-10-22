import 'package:flutter/material.dart';

import 'app_dependencies.dart';
import 'core/theme/app_theme.dart';
import 'features/auth/auth_controller.dart';
import 'features/auth/screens/login_screen.dart';
import 'features/auth/screens/session_failure_screen.dart';
import 'features/auth/screens/startup_screen.dart';
import 'features/documents/controllers/document_upload_controller.dart';
import 'features/documents/documents_controller.dart';
import 'features/groups/groups_controller.dart';
import 'features/home/screens/home_screen.dart';
import 'features/subgroups/subgroups_controller.dart';

class DocProjectApp extends StatefulWidget {
  const DocProjectApp({
    this.dependencies,
    this.authController,
    this.groupsController,
    this.subgroupsController,
    this.documentsController,
    this.uploadController,
    super.key,
  });

  final AppDependencies? dependencies;
  final AuthController? authController;
  final GroupsController? groupsController;
  final SubgroupsController? subgroupsController;
  final DocumentsController? documentsController;
  final DocumentUploadController? uploadController;

  @override
  State<DocProjectApp> createState() => _DocProjectAppState();
}

class _DocProjectAppState extends State<DocProjectApp> {
  late final AppDependencies _dependencies;
  late final AuthController _authController;
  late final GroupsController _groupsController;
  late final SubgroupsController _subgroupsController;
  late final DocumentsController _documentsController;
  late final DocumentUploadController _uploadController;

  @override
  void initState() {
    super.initState();

    _dependencies =
        widget.dependencies ??
        AppDependencies.create(
          authController: widget.authController,
          groupsController: widget.groupsController,
          subgroupsController: widget.subgroupsController,
          documentsController: widget.documentsController,
          uploadController: widget.uploadController,
        );

    _authController = widget.authController ?? _dependencies.authController;
    _groupsController =
        widget.groupsController ?? _dependencies.groupsController;
    _subgroupsController =
        widget.subgroupsController ?? _dependencies.subgroupsController;
    _documentsController =
        widget.documentsController ?? _dependencies.documentsController;
    _uploadController =
        widget.uploadController ?? _dependencies.uploadController;

    // Trigger one-time session restoration on application startup
    _authController.restoreSession();
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'DocProject',
      debugShowCheckedModeBanner: false,
      theme: AppTheme.light,
      home: ListenableBuilder(
        listenable: _authController,
        builder: (context, _) {
          return switch (_authController.state) {
            AuthInitial() || AuthLoading() => const StartupScreen(),
            Authenticated(:final user) => HomeScreen(
              user: user,
              authController: _authController,
              groupsController: _groupsController,
              subgroupsController: _subgroupsController,
              documentsController: _documentsController,
              uploadController: _uploadController,
            ),
            AuthSessionCheckFailure(:final message) => SessionFailureScreen(
              message: message,
              onRetry: () => _authController.restoreSession(force: true),
              onContinueToLogin: _authController.markUnauthenticated,
            ),
            Unauthenticated() => LoginScreen(authController: _authController),
          };
        },
      ),
    );
  }
}
