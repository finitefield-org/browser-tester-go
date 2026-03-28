package browsertester

import (
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func runIntegrationCases(t *testing.T, fns ...func(*testing.T)) {
	t.Helper()
	for _, fn := range fns {
		fn := fn
		t.Run(integrationCaseName(fn), fn)
	}
}

func integrationCaseName(fn func(*testing.T)) string {
	if fn == nil {
		return "nil"
	}

	name := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		name = name[idx+1:]
	}
	return strings.TrimPrefix(name, "Test")
}

func TestIntegrationSuite(t *testing.T) {
	t.Run("debug_parse", func(t *testing.T) {
		runIntegrationCases(t,
			TestDebugParseSingleReportsBootstrapError,
			TestDebugParseFocusActiveElementTernary,
			TestDebugParseActiveElementTernaryDirect,
			TestDebugParseConcatAndTernary,
			TestDebugParseTernaryVariable,
			TestDebugParseWhileLoop,
			TestDebugParseForLoop,
			TestDebugParseIfBlockAndNextStatementWithoutSemicolon,
			TestDebugParseParenthesizedFormulaTrace,
			TestDebugParseRecursiveClosureStopCharAndIndex,
		)
	})

	t.Run("issue_134_137_finitefield_site_regressions", func(t *testing.T) {
		runIntegrationCases(t,
			TestIssue134FinitefieldSiteRegressionObjectAssignGlobalIsAvailable,
			TestIssue135FinitefieldSiteRegressionOptionalChainingListenerOnMemberPathParsesAndRuns,
			TestIssue134FinitefieldSiteRegressionObjectAssignReturnsTargetAndIgnoresNullishSources,
			TestIssue134FinitefieldSiteRegressionObjectAssignCopiesSymbolAndStringSourceKeys,
			TestIssue134FinitefieldSiteRegressionObjectAssignUsesGettersAndSetters,
			TestIssue134FinitefieldSiteRegressionObjectAssignWrapsPrimitiveTargetAndRejectsNullTarget,
			TestIssue136FinitefieldSiteRegressionHtmlButtonElementGlobalSupportsInstanceofChecks,
			TestIssue137FinitefieldSiteRegressionToFixedChainParsesAfterEscapeNormalizationWithUnicode,
		)
	})

	t.Run("issue_138_141_finitefield_site_regressions", func(t *testing.T) {
		runIntegrationCases(t,
			TestIssue138GenericFormatTwoArgsIsNotHijackedAsIntlRelativeTime,
			TestIssue139FunctionCanReferenceLaterDeclaredConst,
			TestIssue140NestedStatePathsAreNotTreatedAsDomElementVariables,
			TestIssue141DispatchKeyboardBubblesToDelegatedListener,
		)
	})

	t.Run("issue_151_finitefield_site_regressions", func(t *testing.T) {
		runIntegrationCases(t,
			TestIssue151MapDeleteWithExtraArgumentIsNotDispatchedAsFormData,
			TestIssue151MapHasWithExtraArgumentUsesMapSemantics,
			TestIssue151PickMapGetOrFallbackDoesNotOverwriteMapBinding,
			TestIssue151PickMapGetOrObjectLiteralFallbackKeepsMap,
			TestIssue151NestedConstShadowNamedPickDoesNotOverwriteOuterPick,
			TestIssue151PlainObjectGetMethodIsNotHijackedAsFormData,
			TestIssue151MapGetPropertyAccessIsNotTreatedAsFormDataCall,
		)
	})

	t.Run("issue_153_154_156_157_159_160_finitefield_site_regressions", func(t *testing.T) {
		runIntegrationCases(t,
			TestIssue153DynamicIndexCompoundAssignmentIsSupported,
			TestIssue154FunctionListenerBindsThisToCurrentTarget,
			TestIssue156RequestAnimationFrameIgnoresExtraArguments,
			TestIssue157DateToLocaleDateStringIsAvailable,
			TestIssue159AssignmentThroughCallResultIsSupported,
			TestIssue160ArrayFlatMapIsSupported,
		)
	})

	t.Run("issue_155_158_finitefield_site_regressions", func(t *testing.T) {
		runIntegrationCases(t,
			TestIssue155ClosestAcceptsSelectorVariableInIfCondition,
			TestIssue158ClosestAcceptsSelectorVariableInExpressionPosition,
		)
	})

	t.Run("issue_165_166_finitefield_site_regressions", func(t *testing.T) {
		runIntegrationCases(t,
			TestIssue165QuotedNewlineSeparatorInJoinIsSupported,
			TestIssue165BuildCsvKeepsRowBreaksBeforeDownload,
			TestIssue165ChainedMapJoinKeepsExplicitSeparator,
			TestIssue165JoinAfterStoringMapResultKeepsExplicitSeparator,
			TestIssue165NamedMapCallbackFollowedByJoinKeepsExplicitSeparator,
			TestIssue165NestedArrayRowsMapToStringsThenJoinWithNewlines,
			TestIssue166MultilineCSVBlobDownloadKeepsRowBreaks,
			TestIssue166InvalidQueryFallbackKeepsDefaultAreaResult,
		)
	})

	t.Run("issue_167_finitefield_site_regressions", func(t *testing.T) {
		runIntegrationCases(t, TestIssue167ReassignedIntlNumberFormatIsUsedByPageCode)
	})

	t.Run("issue_168_finitefield_site_regressions", func(t *testing.T) {
		runIntegrationCases(t, TestIssue168ObjectFromEntriesSupportsPageInitLookupTables)
	})

	t.Run("issue_170_finitefield_site_regressions", func(t *testing.T) {
		runIntegrationCases(t, TestIssue170DOMParserSupportsSvgMime)
	})

	t.Run("issue_171_finitefield_site_regressions", func(t *testing.T) {
		runIntegrationCases(t, TestIssue171IntlCollatorNumericOptionOrdersDigitRunsNaturally)
	})

	t.Run("issue_173_finitefield_site_regressions", func(t *testing.T) {
		runIntegrationCases(t, TestIssue173SwedishCollationOrdersARingBeforeAUmlaut)
	})

	t.Run("issue_174_175_finitefield_site_regressions", func(t *testing.T) {
		runIntegrationCases(t,
			TestIssue174DateTimeFormatAcceptsChicagoAndFormatsCrossZoneResults,
			TestIssue175DateTimeFormatSurfacesNewYorkDstNonexistentAndAmbiguousTimes,
		)
	})

	t.Run("issue_181_finitefield_site_regressions", func(t *testing.T) {
		runIntegrationCases(t,
			TestIssue181XMLSerializerIsAvailableForElementNodes,
			TestIssue181XMLSerializerSerializesSvgAfterDomParserRoundtrip,
		)
	})

	t.Run("issue_183_184_finitefield_site_regressions", func(t *testing.T) {
		runIntegrationCases(t,
			TestIssue183DOMParserReportsParserErrorForMalformedSVG,
			TestIssue184SvgImageHrefAttributesSurviveCloneAndIteration,
		)
	})

	t.Run("issue_185_finitefield_site_regressions", func(t *testing.T) {
		runIntegrationCases(t,
			TestIssue185InlineObjectLiteralComputedLookupReturnsSelectedValue,
			TestIssue185InlineObjectLiteralLookupSurvivesTemplateInterpolation,
		)
	})

	t.Run("issue_190_191_192_finitefield_site_regressions", func(t *testing.T) {
		runIntegrationCases(t,
			TestIssue190DocumentActiveElementTagNameIsSupported,
			TestIssue191DataUrlAnchorDownloadIsCapturedAsArtifact,
			TestIssue192ArrayFlatIsSupported,
			TestIssue192ArrayFlatHonorsDepthAndSkipsSparseSlots,
		)
	})

	t.Run("issue_193_194_finitefield_site_regressions", func(t *testing.T) {
		runIntegrationCases(t,
			TestIssue193PostfixIncrementInsideExpressionIsSupported,
			TestIssue194ArrayDestructureAssignmentInsideElseIfBranchIsSupported,
		)
	})

	t.Run("issue_199_finitefield_site_regression", func(t *testing.T) {
		runIntegrationCases(t, TestIssue199RepeatedTypeTextWithLargeIDHeavyDOMAndHistoryStorageSyncCompletes)
	})

	t.Run("issue_202_finitefield_site_regressions", func(t *testing.T) {
		runIntegrationCases(t,
			TestIssue202AsyncDigestStubUpdatesDomAfterAwait,
			TestIssue202WindowPropertyReadsAsGlobalIdentifierInsideFunction,
		)
	})

	t.Run("issue_203_finitefield_site_regressions", func(t *testing.T) {
		runIntegrationCases(t,
			TestIssue203StickyElementStaysPinnedAfterWindowScroll,
			TestIssue203StickyElementHonorsRemTopInsetDuringScroll,
		)
	})

	t.Run("issue_211_finitefield_site_regressions", func(t *testing.T) {
		runIntegrationCases(t,
			TestIssue211DumpDOMPreservesAdjustedSVGAttributeCasing,
			TestIssue211DumpDOMDoesNotRecaseHTMLAttributesOutsideSVG,
		)
	})

	t.Run("issue_212_finitefield_site_regressions", func(t *testing.T) {
		runIntegrationCases(t,
			TestIssue212TypeofWindowHistoryReplaceStateMemberReference,
			TestIssue212TypeofWindowLocationReplaceMemberReference,
		)
	})

	t.Run("issue_212_finitefield_site_runtime_regressions", func(t *testing.T) {
		runIntegrationCases(t,
			TestIssue212NestedPendingHelperInListenerKeepsOuterStateCapture,
			TestIssue212HostLikeRenderChainInListenerKeepsOuterStateCapture,
			TestIssue212BulkMappingListenerCanReplaceOuterConstStateArray,
			TestIssue212BulkCallbackCanUpdateOuterLetCounterInsideListenerRender,
		)
	})

	t.Run("issue_214_array_map_outer_let_regression", func(t *testing.T) {
		runIntegrationCases(t, TestIssue214ArrayMapCallbackMutationsUpdateOuterLetBindings)
	})

	t.Run("issue_215_nested_helper_const_leak_regression", func(t *testing.T) {
		runIntegrationCases(t,
			TestIssue215NestedHelperLocalIndexDoesNotPoisonLaterConstIndex,
			TestIssue215NestedHelperLocalIndexDoesNotPoisonPlainConstDeclaration,
		)
	})

	t.Run("issue_217_return_slot_regression", func(t *testing.T) {
		runIntegrationCases(t,
			TestIssue217NestedHelperCallInReturnExpressionKeepsOuterReturnValue,
			TestIssue217BatchMappingGridKeepsSelectMarkupWithLateHelperDeclaration,
		)
	})

	t.Run("issue_218_bulk_callback_const_binding_regression", func(t *testing.T) {
		runIntegrationCases(t,
			TestIssue218BulkMappingAndSummaryCallbacksKeepOuterBindingsIsolatedAndAccumulating,
			TestIssue218BulkResultMapAccumulatesMixedPriceAndTargetRows,
			TestIssue218NestedBulkParserHelpersKeepOuterRowsAndCellsLive,
			TestIssue218NestedHelperUpdatesStayVisibleAfterDirectHelperCall,
			TestIssue218ArrayCallbackCounterSurvivesPlainHelperCalls,
			TestIssue218ArrayCallbackUpdatesRemainVisibleToLaterPlainHelperCalls,
			TestIssue218HelperUpdatesAreVisibleToLaterArrayMapInSameFunction,
			TestIssue218RepeatedPushCellUpdatesFeedParserStylePushRow,
			TestIssue218SimpleLoopParserKeepsPushCellAndPushRowUpdates,
			TestIssue218SimpleLoopPushCellUpdatesSurviveAcrossContinueIterations,
		)
	})

	t.Run("issue_219_finitefield_site_regressions", func(t *testing.T) {
		runIntegrationCases(t, TestIssue219MarginMarkupPageLoadsAndRunsBulkFlowOnSmallTestThread)
	})

	t.Run("open_issue_regressions", func(t *testing.T) {
		runIntegrationCases(t,
			TestClickTogglesButtonInsideOpenDialog,
			TestWindowOpenReturnsPopupStubForPrintFlows,
			TestClickPreservesPreRequestAnimationFrameProcessingState,
			TestDispatchKeyboardCompletesAsyncKeydownHandlersWaitingForAnimationFrame,
			TestIIFEHelperListenerReadsLiveOuterLetAfterSiblingRender,
			TestIIFEFunctionKeepsLaterSiblingFunctionDeclarationCallable,
			TestListenerReferenceKeepsPendingFunctionDeclOuterCapture,
			TestSiblingClosureCallsDoNotPruneScopeCaptureEnv,
			TestNestedCallPreservesCallerLocalCloseBinding,
			TestNestedCallKeepsCallerLocalBindingBeforeFollowUpCalls,
			TestNestedCallKeepsCallerLocalBindingAfterSiblingCall,
			TestTrivialNestedCallDoesNotReplaceLocalCloseWithWindowClose,
			TestNestedCallKeepsCapturedIndexVisibleToBareReads,
			TestNestedParseNumberKeepsOuterProgressVisible,
			TestPlainFormulaParserAcceptsParenthesizedGroups,
			TestForeachAttachedClickHandlerReassignsOuterState,
			TestForeachAttachedClickHandlerReassignsOuterStateInIIFEPageFlow,
			TestForOfLoopSupportsArrayDestructuringBinding,
			TestAppendChildSyncsSelectValueForPreselectedOption,
			TestNestedHelperFunctionRetainsTransitiveOuterCapture,
			TestClassFieldInitializerKeepsOuterCaptureThroughFactory,
		)
	})

	t.Run("regression_runtime_state_fixes", func(t *testing.T) {
		runIntegrationCases(t, TestRegressionRuntimeStateFixesRecursiveConstArrowClosureCanReferenceItself)
	})

	t.Run("typed_array_from_map_fn", func(t *testing.T) {
		runIntegrationCases(t, TestTypedArrayFromSupportsMapFunctionArgument)
	})
}
