package dev.unqueryvet.actions

import com.intellij.openapi.actionSystem.AnAction
import com.intellij.openapi.actionSystem.AnActionEvent
import com.intellij.openapi.actionSystem.CommonDataKeys
import com.intellij.openapi.command.WriteCommandAction
import com.intellij.openapi.progress.ProgressIndicator
import com.intellij.openapi.progress.ProgressManager
import com.intellij.openapi.progress.Task
import com.intellij.openapi.ui.Messages
import com.intellij.psi.PsiManager
import com.intellij.psi.util.PsiTreeUtil
import dev.unqueryvet.UnqueryvetSettings
import java.io.BufferedReader
import java.io.File
import java.io.InputStreamReader

/**
 * Action to fix all SELECT * issues in the current file or project.
 */
class FixAllAction : AnAction() {

    override fun actionPerformed(e: AnActionEvent) {
        val project = e.project ?: return
        val file = e.getData(CommonDataKeys.VIRTUAL_FILE) ?: return

        // Confirm with user
        val result = Messages.showYesNoDialog(
            project,
            "This will replace all SELECT * with TODO comments. Continue?",
            "Fix All SELECT *",
            Messages.getQuestionIcon()
        )

        if (result != Messages.YES) {
            return
        }

        ProgressManager.getInstance().run(object : Task.Backgroundable(project, "Fixing SELECT * issues") {
            override fun run(indicator: ProgressIndicator) {
                indicator.isIndeterminate = true
                indicator.text = "Analyzing..."

                try {
                    val settings = UnqueryvetSettings.getInstance()
                    val command = mutableListOf(
                        settings.binaryPath,
                        "-fix",
                        file.path
                    )

                    val process = ProcessBuilder(command)
                        .directory(File(project.basePath ?: "."))
                        .redirectErrorStream(true)
                        .start()

                    val reader = BufferedReader(InputStreamReader(process.inputStream))
                    val output = reader.readText()
                    reader.close()

                    val exitCode = process.waitFor()

                    com.intellij.openapi.application.ApplicationManager.getApplication().invokeLater {
                        if (exitCode == 0) {
                            // Refresh file
                            file.refresh(false, false)

                            Messages.showInfoMessage(
                                project,
                                "Fixed all SELECT * issues",
                                "Fix Complete"
                            )
                        } else {
                            Messages.showWarningDialog(
                                project,
                                "Some issues could not be fixed automatically.\n$output",
                                "Fix Incomplete"
                            )
                        }
                    }

                } catch (ex: Exception) {
                    com.intellij.openapi.application.ApplicationManager.getApplication().invokeLater {
                        Messages.showErrorDialog(
                            project,
                            "Error fixing issues: ${ex.message}",
                            "Fix Failed"
                        )
                    }
                }
            }
        })
    }

    override fun update(e: AnActionEvent) {
        val project = e.project
        val file = e.getData(CommonDataKeys.VIRTUAL_FILE)
        e.presentation.isEnabledAndVisible = project != null && file != null && file.extension == "go"
    }
}
